package gateways

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hexablock/fidias"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
)

// HTTPServer serves all http requests that guac supports
type HTTPServer struct {
	prefix string     // api version prefix
	routes httpRoutes // Registerd routes

	conf     *fidias.Config
	ring     *hexaring.Ring
	fids     *fidias.Fidias
	fsm      fidias.KeyValueFSM
	logstore hexalog.LogStore
}

// NewHTTPServer instantiates a new Guac HTTP API server
func NewHTTPServer(apiPrefix string, conf *fidias.Config, ring *hexaring.Ring, fsm fidias.KeyValueFSM, logstore hexalog.LogStore, fids *fidias.Fidias) *HTTPServer {
	s := &HTTPServer{
		prefix:   apiPrefix,
		conf:     conf,
		ring:     ring,
		fsm:      fsm,
		logstore: logstore,
		fids:     fids,
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
		data = server.fids.Status()
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

// handleLookup lookups the requested number of successors n.  If n is not provied it is
// defaulted to the max no. of allowed successorss
func (server *HTTPServer) handleLookup(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {

	var n int
	if n, err = parseIntQueryParam(r, "n"); err != nil {
		return
	}
	if n == 0 {
		n = server.ring.NumSuccessors()
	}

	code = 200
	_, data, err = server.ring.Lookup(n, []byte(resourceID))

	return
}
