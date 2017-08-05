package fidias

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hexablock/hexalog"
)

// handleHexalog serves http requests to get a keylog and add an entry to the keylog by key
func (server *HTTPServer) handleHexalog(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {
	headers = map[string]string{}

	if resourceID == "" {
		code = 404
		return
	}

	code = 200

	switch r.Method {
	case http.MethodGet:
		keid := strings.Split(resourceID, "/")
		// Get a secific entry for a key
		if len(keid) == 2 {
			var id []byte
			if id, err = hex.DecodeString(keid[1]); err == nil {
				key := []byte(keid[0])
				meta := &ReMeta{}
				if data, meta, err = server.fidias.GetEntry(key, id); err != nil {
					code = 404
				} else {
					headers[headerLocations] = fmt.Sprintf("%s/%x", meta.Vnode.Host, meta.Vnode.Id)
				}
			}
		} else {
			// Get the complete log.
			data, err = server.logstore.GetKey([]byte(resourceID))
		}

	case http.MethodPost:
		// Append an entry to the keylog
		var b []byte
		if b, err = ioutil.ReadAll(r.Body); err != nil {
			break
		}
		defer r.Body.Close()

		entry := server.fidias.NewEntry([]byte(resourceID))
		entry.Data = b

		var (
			ballot *hexalog.Ballot
			meta   *ReMeta
		)
		if ballot, meta, err = server.fidias.ProposeEntry(entry); err == nil {
			if err = ballot.Wait(); err == nil {
				data = ballot.Future()
				headers[headerLocations] = locationSetHeaderVals(meta.PeerSet)
			}
			headers[headerBallotTime] = fmt.Sprintf("%v", ballot.Runtime())

		} else if strings.Contains(err.Error(), "not in peer set") {
			// Redirect to the natural key holder
			if data, err = generateRedirect(meta.PeerSet[0].Vnode, r.RequestURI); err == nil {
				code = statusCodeRedirect
			}

		}

	case http.MethodOptions:
		headers["Content-Type"] = contentTypeTextPlain
		data = server.hexalogOptionsBody(resourceID)

	default:
		code = 405
	}

	return
}

func (server *HTTPServer) hexalogOptionsBody(resourceID string) []byte {
	return []byte(fmt.Sprintf(`
  %s/hexalog/%s

  Endpoint to perform direct hexalog entry operations

  Methods:

    GET      Retrieve the key log for key: '%s'
    POST     Append an entry to the key log for key: '%s'
    OPTIONS  Information about the endpoint

  Body:

    Key log entry data

`, server.prefix, resourceID, resourceID, resourceID))
}
