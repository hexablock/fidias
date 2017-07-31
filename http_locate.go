package fidias

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
			n = server.fidias.conf.Replicas
		}

		start := time.Now()
		code = 200
		data, err = server.fidias.ring.LookupReplicated([]byte(resourceID), n)
		end := time.Since(start)
		headers[headerLocateTime] = fmt.Sprintf("%v", end)

	case http.MethodOptions:
		code = 200
		headers["Content-Type"] = contentTypeTextPlain
		data = server.locateOptionsBody(resourceID)

	default:
		code = 405

	}

	return
}

func (server *HTTPServer) locateOptionsBody(resourceID string) []byte {
	return []byte(fmt.Sprintf(`
  %s/locate/%s

  Endpoint to perform key location operations

  Methods:

    GET      Retreives replicated locations for key: '%s'
    OPTIONS  Information about the endpoint

  Parameters:

    r        Number of replicated locations

`, server.prefix, resourceID, resourceID))
}

// handleLookup lookups the requested number of successors n.  If n is not provied it is
// defaulted to the max no. of allowed successorss
func (server *HTTPServer) handleLookup(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {

	var n int
	if n, err = parseIntQueryParam(r, "n"); err != nil {
		return
	}
	if n == 0 {
		n = server.fidias.ring.NumSuccessors()
	}

	code = 200
	_, data, err = server.fidias.ring.Lookup(n, []byte(resourceID))

	return
}
