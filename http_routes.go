package fidias

import "net/http"

// httpHandlerFunc is the handler function for a given path.  Handlers need to implement
// this definition for each handler.  It returns the code, headers, data and an error if any
type httpHandlerFunc func(http.ResponseWriter, *http.Request, string) (int, map[string]string, interface{}, error)

type httpRoutes map[string]httpHandlerFunc

// handler returns the handler function and resource id based on the given input resource.
// It returns the handler and resource id if found otherwise nil is returned for the
// handler
func (r httpRoutes) handler(s string) (httpHandlerFunc, string) {
	var resource, resourceID string

	// Parse input to get resource id
	for i, c := range s {
		l := len(s)

		if i == l-1 {
			resource = s
			break
		} else if c == '/' {
			resource = s[:i]
			if i < l {
				resourceID = s[i+1:]
			}
			break
		}
	}

	// Empty resources are not allowed
	if resource == "" {
		return nil, resourceID
	}

	// Get the handler
	handler, _ := r[resource]

	return handler, resourceID
}
