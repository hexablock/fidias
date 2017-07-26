package fidias

import (
	"fmt"
	"net/http"
)

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

		code = 200
		data, err = server.fidias.ring.LookupReplicated([]byte(resourceID), n)

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
