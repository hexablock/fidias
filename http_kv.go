package fidias

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
)

func (server *HTTPServer) handleGet(resourceID string, n int, reqURI string) (code int, data interface{}, err error) {
	var (
		key  = []byte(resourceID)
		locs hexaring.LocationSet
	)

	if locs, err = server.fidias.ring.LookupReplicated(key, n); err != nil {
		return
	}

	host := server.fidias.conf.Hostname()

	// Check if host is part of the location set otherwise re-direct to the natural vnode
	//var loc *hexaring.Location
	if _, err = locs.GetByHost(host); err != nil {
		if data, err = generateRedirect(locs[0].Vnode, reqURI); err == nil {
			code = statusCodeRedirect
		}
		return
	}

	data, _, err = server.fidias.GetKey(key)
	if err == nil {
		code = 200
	} else {
		if strings.Contains(err.Error(), "not found") {
			code = 404
		}
	}

	return
}

func (server *HTTPServer) handleKeyValue(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {
	headers = map[string]string{}

	if resourceID == "" {
		code = 404
		return
	}

	code = 200

	switch r.Method {
	case http.MethodGet:
		var n int
		n, err = parseIntQueryParam(r, "n")
		if err != nil {
			break
		}
		if n == 0 {
			n = server.fidias.conf.Replicas
		}

		code, data, err = server.handleGet(resourceID, n, r.RequestURI)

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
				headers[headerLocations] = locationSetHeaderVals(meta.PeerSet)
			}
			headers[headerBallotTime] = fmt.Sprintf("%v", ballot.Runtime())

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

	// case http.MethodOptions:
	// 	code = 200
	// 	headers["Content-Type"] = contentTypeTextPlain
	// 	data = server.kvOptionsBody(resourceID)

	default:
		code = 405
	}

	return
}

// func (server *HTTPServer) kvOptionsBody(resourceID string) []byte {
// 	return []byte(fmt.Sprintf(`
//   %s/kv/%s
//
//   Endpoint to perform key-value operations
//
//   Methods:
//
//     GET      Return value for key: '%s'
//     POST     Create or update key: '%s'
//     DELETE   Delete key: '%s'
//     OPTIONS  Information about the endpoint
//
//   Body:
//
//     Arbitrary data associated to the key
//
// `, server.prefix, resourceID, resourceID, resourceID, resourceID))
// }
