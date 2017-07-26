package fidias

import (
	"encoding/json"
	"net/http"
	"strconv"
)

const contentTypeTextPlain = "text/plain"

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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	for k, v := range headers {
		w.Header().Set(k, v)
	}

	w.WriteHeader(c)
	w.Write(b)
}
