package fidias

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hexablock/hexalog"
)

func (server *HTTPServer) handleKeyValue(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {
	headers = map[string]string{}

	if resourceID == "" {
		code = 404
		return
	}

	code = 200

	switch r.Method {
	case http.MethodGet:
		keid := strings.Split(resourceID, "/")
		// Get specific version of a key by entry id
		if len(keid) == 2 {
			var id []byte
			if id, err = hex.DecodeString(keid[1]); err == nil {
				key := []byte(keid[0])
				var entry *hexalog.Entry
				if entry, _, err = server.fidias.GetEntry(key, id); err != nil {
					code = 404
				} else {
					data = &KeyValueItem{Key: keid[0], Entry: entry, Value: entry.Data[1:]}
				}
			}
		} else {
			// Get a key
			if val := server.fsm.Get(resourceID); val != nil {
				data = val
			} else {
				code = 404
			}
		}

	case http.MethodPost:
		// Append a set operation entry to the log
		var b []byte
		if b, err = ioutil.ReadAll(r.Body); err != nil {
			break
		}
		defer r.Body.Close()

		entry := server.fidias.NewEntry([]byte(resourceID))
		entry.Data = append([]byte{opSet}, b...)
		code = 200

		var (
			ballot *hexalog.Ballot
			meta   *ReMeta
		)
		if ballot, meta, err = server.fidias.ProposeEntry(entry); err == nil {
			if err = ballot.Wait(); err == nil {
				data = ballot.Future()
			}
		} else if strings.Contains(err.Error(), "not in peer set") {
			// Redirect to the natural key holder
			if data, err = generateRedirect(meta.PeerSet[0].Vnode, r.RequestURI); err == nil {
				code = statusCodeRedirect
			}

		}

	case http.MethodDelete:
		// Append a delete operation entry to the log
		entry := server.fidias.NewEntry([]byte(resourceID))
		entry.Data = []byte{opDel}
		code = 200

		var (
			ballot *hexalog.Ballot
			meta   *ReMeta
		)
		if ballot, meta, err = server.fidias.ProposeEntry(entry); err == nil {
			if err = ballot.Wait(); err == nil {
				data = ballot.Future()
			}
		} else if strings.Contains(err.Error(), "not in peer set") {
			// Redirect to the natural key holder
			if data, err = generateRedirect(meta.PeerSet[0].Vnode, r.RequestURI); err == nil {
				code = statusCodeRedirect
			}
		}

	case http.MethodOptions:
		code = 200
		headers["Content-Type"] = contentTypeTextPlain
		data = server.kvOptionsBody(resourceID)

	default:
		code = 405
	}

	return
}

func (server *HTTPServer) kvOptionsBody(resourceID string) []byte {
	return []byte(fmt.Sprintf(`
  %s/kv/%s

  Endpoint to perform key-value operations

  Methods:

    GET      Return value for key: '%s'
    POST     Create or update key: '%s'
    DELETE   Delete key: '%s'
    OPTIONS  Information about the endpoint

  Body:

    Arbitrary data associated to the key

`, server.prefix, resourceID, resourceID, resourceID, resourceID))
}
