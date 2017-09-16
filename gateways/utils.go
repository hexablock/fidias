package gateways

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hexablock/hexaring"
)

const contentTypeTextPlain = "text/plain"

const (
	// headerLocations is the header key for locations
	headerLocations = "Location-Set"
	// headerBallotTime is the header key for the time taken for a ballot
	headerBallotTime = "Ballot-Time"
	// headerLocateTime is the header key for the time taken for a locate call
	headerLocateTime = "Locate-Time"
	headerLookupTime = "Lookup-Time"
	headerBlockSize  = "Block-Size"
	headerRuntime    = "Runtime"
)

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

// writeJSONResponse writes a json response.  It first sets the headers, then the code and
// finally the data.  It manages serializing the data.  It data is a byte slice then it simply
// writes the data without setting any content type headers
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

	// Write code
	w.WriteHeader(c)
	// Write data
	w.Write(b)
}

func buildRedirect(locs hexaring.LocationSet, reqPath string) (code int, headers map[string]string, data interface{}, err error) {
	headers = make(map[string]string)

	// simply return out error
	// if !strings.Contains(e.Error(), "host not in set") {
	// 	err = e
	// 	return
	// }

	loc := locs[0]
	meta := loc.Vnode.Metadata()

	host, ok := meta["http"]
	if !ok {
		err = fmt.Errorf("Meta.http host not found")
		return
	}

	headers["Location"] = fmt.Sprintf("http://%s%s", host, reqPath)
	code = 307
	data = loc

	return
}
