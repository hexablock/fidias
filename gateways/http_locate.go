package gateways

import (
	"fmt"
	"net/http"
	"time"
)

// handleLocate locates r replicas around the ring.  If r is not provided then it is
// defaulted to the number of replicas in the configuration
func (server *HTTPServer) handleLocate(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {
	headers = map[string]string{}

	switch r.Method {
	case http.MethodGet:
		// Parse requested replicas
		var n int
		if n, err = parseIntQueryParam(r, "r"); err != nil {
			return
		}
		// Set default if not provided
		if n == 0 {
			n = server.conf.Hexalog.Votes
		}

		code = 200

		start := time.Now()
		data, err = server.ring.LookupReplicated([]byte(resourceID), n)
		end := time.Since(start)

		headers[headerLocateTime] = fmt.Sprintf("%v", end)

	default:
		code = 405

	}

	return
}
