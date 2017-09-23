package gateways

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hexablock/blox/filesystem"
	"github.com/hexablock/fidias"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
)

// HTTPServer serves all http requests that guac supports
type HTTPServer struct {
	prefix string     // api version prefix
	routes httpRoutes // Registerd routes

	conf *fidias.Config
	ring *hexaring.Ring
	fids *fidias.Fidias

	logstore *hexalog.LogStore

	keyvs *fidias.Keyvs

	dev filesystem.BlockDevice
	fs  *filesystem.BloxFS
}

// NewHTTPServer instantiates a new Guac HTTP API server
func NewHTTPServer(apiPrefix string, conf *fidias.Config, ring *hexaring.Ring, keyvs *fidias.Keyvs, logstore *hexalog.LogStore, dev filesystem.BlockDevice, fids *fidias.Fidias) *HTTPServer {
	s := &HTTPServer{
		prefix:   apiPrefix,
		conf:     conf,
		ring:     ring,
		logstore: logstore,
		fids:     fids,
		keyvs:    keyvs,
		dev:      dev,
		fs:       filesystem.NewBloxFS(dev),
	}

	// Register URL path to handler
	s.routes = httpRoutes{
		"locate":  s.handleLocate,   // Replicated lookups
		"lookup":  s.handleLookup,   // Chord lookups
		"hexalog": s.handleHexalog,  // Hexalog interations
		"kv":      s.handleKeyValue, // Key-value operations
		"status":  s.handleStatus,   // Overall status
	}

	return s
}

func (server *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqpath := strings.TrimPrefix(r.URL.Path, server.prefix)
	reqpath = strings.TrimPrefix(reqpath, "/")

	// Blox handler.  This is dealt with here as it has a different handler
	// definition then the rest
	if strings.HasPrefix(reqpath, "blox") {
		server.handleBlox(w, r, strings.TrimPrefix(reqpath, "blox/"))
		return
	} else if strings.HasPrefix(reqpath, "fs") {
		server.handleFS(w, r, strings.TrimPrefix(reqpath, "fs/"))
		return
	}

	handler, resourceID := server.routes.handler(reqpath)
	if handler == nil {
		w.WriteHeader(404)
		return
	}

	code, headers, data, err := handler(w, r, resourceID)
	writeJSONResponse(w, code, headers, data, err)
}

func (server *HTTPServer) handleStatus(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {
	code = 200
	data = server.fids.Status()
	return
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
	start := time.Now()
	_, data, err = server.ring.Lookup(n, []byte(resourceID))
	end := time.Since(start)
	headers = map[string]string{headerLookupTime: fmt.Sprintf("%v", end)}

	return
}
