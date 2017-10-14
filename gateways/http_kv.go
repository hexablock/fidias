package gateways

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/hexablock/hexalog"
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
	headers = make(map[string]string)

	var meta *hexalog.RequestOptions

	switch r.Method {
	case http.MethodGet:
		start := time.Now()
		data, meta, err = server.keyvs.GetKey([]byte(resourceID))
		headers[headerRuntime] = fmt.Sprintf("%v", time.Since(start))

	case http.MethodPost, http.MethodPut:
		start := time.Now()
		// Append a set operation entry to the log
		var b []byte
		if b, err = ioutil.ReadAll(r.Body); err != nil {
			break
		}
		defer r.Body.Close()

		data, meta, err = server.keyvs.SetKey([]byte(resourceID), b)
		headers[headerRuntime] = fmt.Sprintf("%v", time.Since(start))

	case http.MethodDelete:
		start := time.Now()
		data, meta, err = server.keyvs.RemoveKey([]byte(resourceID))
		headers[headerRuntime] = fmt.Sprintf("%v", time.Since(start))

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
