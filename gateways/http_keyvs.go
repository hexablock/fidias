package gateways

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hexablock/hexatype"
)

func (server *HTTPServer) handleKeyValue(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {
	if resourceID == "" {
		code = 404
		return
	}

	// Parameters
	// var n int
	// if n, err = parseIntQueryParam(r, "n"); err != nil {
	// 	return
	// }
	// if n == 0 {
	// 	n = server.conf.Hexalog.Votes
	// }

	code = 200

	var meta *hexatype.RequestOptions

	switch r.Method {
	case http.MethodGet:
		data, meta, err = server.keyvs.GetKey([]byte(resourceID))

	case http.MethodPost, http.MethodPut:
		// Append a set operation entry to the log
		var b []byte
		if b, err = ioutil.ReadAll(r.Body); err != nil {
			break
		}
		defer r.Body.Close()

		data, meta, err = server.keyvs.SetKey([]byte(resourceID), b)

	case http.MethodDelete:
		data, meta, err = server.keyvs.RemoveKey([]byte(resourceID))

	default:
		code = 405
		return
	}

	if err == nil {
		return
	}

	if err == hexatype.ErrKeyNotFound {
		code = 404
	} else if strings.Contains(err.Error(), "host not in set") {
		code, headers, data, err = buildRedirect(meta.PeerSet, r.URL.RequestURI())
	}

	return
}
