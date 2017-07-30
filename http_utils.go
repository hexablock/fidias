package fidias

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexaring"
)

const contentTypeTextPlain = "text/plain"

// headerLocations is the header key for locations
const headerLocations = "Location-Set"

// statusCodeRedirect will keep the data for the call
const statusCodeRedirect = 307

var accessControlHeaders = map[string]string{
	"Access-Control-Allow-Origin": "*",
}

// locationSetHeader returns the Location-Set header value
func locationSetHeaderVals(peerSet hexaring.LocationSet) string {
	h := make([]string, len(peerSet))
	for i, p := range peerSet {
		h[i] = fmt.Sprintf("%s/%x", p.Vnode.Host, p.Vnode.Id)
	}
	return strings.Join(h, ",")
}

// generateRedirect generates the redirect url based on the given vnode
func generateRedirect(vn *chord.Vnode, reqURI string) (s string, err error) {
	mt := chord.Meta{}
	if err = mt.UnmarshalBinary(vn.Meta); err == nil {
		host, _ := mt["http"]
		s = fmt.Sprintf("http://%s%s", host, reqURI)
	}

	return
}

// parseIntQueryParam parses an int from the url query parameters.  It only uses the first
// element of the param slice
func parseIntQueryParam(r *http.Request, param string) (int, error) {
	var d int

	q := r.URL.Query()
	if qr, ok := q[param]; ok && len(qr) > 0 {
		r, err := strconv.ParseInt(qr[0], 10, 64)
		if err == nil {
			d = int(r)
		}
		return d, err
	}

	return d, nil
}

func writeJSONResponse(w http.ResponseWriter, code int, headers map[string]string, data interface{}, err error) {
	var (
		b []byte
		c = code
	)

	// Make sure the code is > 400 if it is an error otherwise set it to 400
	if err != nil {
		if code < 400 {
			c = 400
		}
		// Error data
		b = []byte(`{"error":"` + err.Error() + `"}`)

	} else {
		if data != nil {
			// Write byte slice directly
			if t, ok := data.([]byte); ok {
				b = t
			} else {
				b, _ = json.Marshal(data)
				w.Header().Set("Content-Type", "application/json")
			}
		}

	}

	// Set headers supplied as input
	for k, v := range headers {
		w.Header().Set(k, v)
	}

	// Set ACL headers after so they are not overwritten by the caller
	for k, v := range accessControlHeaders {
		w.Header().Set(k, v)
	}

	w.WriteHeader(c)
	w.Write(b)
}
