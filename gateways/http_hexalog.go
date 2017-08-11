package gateways

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hexablock/fidias"
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
				meta := &fidias.ReMeta{}
				if data, meta, err = server.fids.GetEntry(key, id); err != nil {
					code = 404
				} else {
					headers[headerLocations] = fmt.Sprintf("%s/%x", meta.Vnode.Host, meta.Vnode.Id)
				}
			}
		} else {
			// Get the keylog index only.
			var lk *hexalog.Keylog
			if lk, err = server.logstore.GetKey([]byte(resourceID)); err == nil {
				data = lk.GetIndex()
			}

		}

	case http.MethodPost:
		// Append an entry to the keylog
		var b []byte
		if b, err = ioutil.ReadAll(r.Body); err != nil {
			break
		}
		defer r.Body.Close()

		entry := server.fids.NewEntry([]byte(resourceID))
		entry.Data = b

		var (
			ballot *hexalog.Ballot
			meta   *fidias.ReMeta
		)
		if ballot, meta, err = server.fids.ProposeEntry(entry); err == nil {
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

	default:
		code = 405
	}

	return
}
