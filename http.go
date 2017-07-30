package fidias

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hexablock/hexalog"
)

// HTTPServer serves all http requests that guac supports
type HTTPServer struct {
	prefix   string     // api version prefix
	routes   httpRoutes // Registerd routes
	fidias   *Fidias
	fsm      *KeyValueFSM
	logstore hexalog.LogStore
}

// NewHTTPServer instantiates a new Guac HTTP API server
func NewHTTPServer(apiPrefix string, fsm *KeyValueFSM, logstore hexalog.LogStore, fidias *Fidias) *HTTPServer {
	s := &HTTPServer{
		prefix:   apiPrefix,
		fsm:      fsm,
		logstore: logstore,
		fidias:   fidias,
	}
	// URL path to handler map
	s.routes = httpRoutes{
		"lookup":  s.handleLookup,   // Chord lookups
		"locate":  s.handleLocate,   // Replicated lookups
		"status":  s.handleStatus,   // Overall status
		"hexalog": s.handleHexalog,  // Hexalog interations
		"kv":      s.handleKeyValue, // Key-value operations
	}

	return s
}

func (server *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqpath := strings.TrimPrefix(r.URL.Path, server.prefix)
	reqpath = strings.TrimPrefix(reqpath, "/")
	handler, resourceID := server.routes.handler(reqpath)

	if handler == nil {
		w.WriteHeader(404)
		return
	}

	code, headers, data, err := handler(w, r, resourceID)
	if code == statusCodeRedirect {
		urlstr := data.(string)
		http.Redirect(w, r, urlstr, code)
		return
	}

	writeJSONResponse(w, code, headers, data, err)
}

func (server *HTTPServer) handleStatus(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {

	switch r.Method {
	case http.MethodGet:
		code = 200
		data = server.fidias.Status()
	case http.MethodOptions:
		code = 200
		data = server.statusOptionsBody(resourceID)
	default:
		code = 405
	}

	return
}

func (server *HTTPServer) statusOptionsBody(resourceID string) []byte {
	return []byte(fmt.Sprintf(`
  %s/status

  Endpoint to gets status information about the node

  Methods:

    GET      Retreives the node status
    OPTIONS  Information about the endpoint

`, server.prefix))
}
