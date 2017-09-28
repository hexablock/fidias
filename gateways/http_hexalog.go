package gateways

import (
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/hexablock/hexatype"
)

// handleHexalog serves http requests to get a keylog and add an entry to the keylog by key
func (server *HTTPServer) handleHexalog(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {
	headers = map[string]string{}

	if resourceID == "" {
		code = 404
		return
	}

	switch r.Method {
	case http.MethodGet:
		keid := strings.Split(resourceID, "/")
		// Handle entry get
		id, er := hex.DecodeString(keid[len(keid)-1])
		if er == nil {
			k := keid[len(keid)-2]
			code = 200
			data, err = server.logstore.GetEntry([]byte(k), id)
			return
		}
		// Handle log index get
		code, data, err = server.handleGetKeylog([]byte(resourceID))

	default:
		code = 405
	}

	return
}

func (server *HTTPServer) handleGetKeylog(key []byte) (int, interface{}, error) {
	keylog, err := server.logstore.GetKey(key)
	if err == nil {
		return 200, keylog.GetIndex(), nil
	}
	if err == hexatype.ErrKeyNotFound {
		return 404, nil, err
	}
	return 400, nil, err
}

func (server *HTTPServer) handleGetEntry(key []byte, idstr string) (int, interface{}, error) {
	id, err := hex.DecodeString(idstr)
	if err != nil {
		return 400, nil, err
	}

	code := 200
	entry, err := server.logstore.GetEntry(key, id)
	if err == hexatype.ErrEntryNotFound {
		code = 404
	}
	return code, entry, err
}
