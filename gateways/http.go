package gateways

import (
	"log"
	"net/http"
	"strings"

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
	reqpath = strings.TrimPrefix(reqpath, "/") // remove leading

	// Blox handler.  This is dealt with here as it has a different handler
	// definition then the rest
	if strings.HasPrefix(reqpath, "blox") {
		server.handleBlox(w, r, strings.TrimPrefix(reqpath, "blox/"))
		return
	}

	// Remove trailing for all remaining handlers
	reqpath = strings.TrimSuffix(reqpath, "/")
	handler, resourceID := server.routes.handler(reqpath)
	if handler == nil {
		w.WriteHeader(404)
		return
	}

	code, headers, data, err := handler(w, r, resourceID)
	writeJSONResponse(w, code, headers, data, err)
	log.Printf("[INFO] %d %s %s", code, r.Method, r.RequestURI)
}

func (server *HTTPServer) handleStatus(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {
	code = 200
	data = server.fids.Status()
	return
}
