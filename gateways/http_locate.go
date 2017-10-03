package gateways

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// handleLocate locates r replicas around the ring.  If r is not provided then it is
// defaulted to the number of replicas in the configuration
func (server *HTTPServer) handleLocate(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {
	headers = map[string]string{}

	// Parse requested replicas
	var n int
	if n, err = parseIntQueryParam(r, "r"); err != nil {
		return
	}
	// Set default if not provided
	if n == 0 {
		n = server.conf.Hexalog.Votes
	}

	start := time.Now()
	sh, err := hex.DecodeString(resourceID)
	if err != nil {
		data, err = server.ring.LookupReplicated([]byte(resourceID), n)
	} else {
		data, err = server.ring.LookupReplicatedHash(sh, n)
	}
	end := time.Since(start)

	code = 200
	headers[headerLocateTime] = fmt.Sprintf("%v", end)

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

	start := time.Now()
	sh, err := hex.DecodeString(resourceID)
	if err != nil {
		_, data, err = server.ring.Lookup(n, []byte(resourceID))
	} else {
		data, err = server.ring.LookupHash(n, sh)
	}
	end := time.Since(start)

	code = 200
	headers = map[string]string{headerLookupTime: fmt.Sprintf("%v", end)}

	return
}
