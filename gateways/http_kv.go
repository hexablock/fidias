package gateways

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hexablock/fidias"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
)

func (server *HTTPServer) handleGet(resourceID string, n int, reqURI string) (code int, data interface{}, err error) {
	var (
		key  = []byte(resourceID)
		locs hexaring.LocationSet
	)

	if locs, err = server.ring.LookupReplicated(key, n); err != nil {
		return
	}

	host := server.conf.Hostname()

	// Check if host is part of the location set otherwise re-direct to the natural vnode
	//var loc *hexaring.Location
	if _, err = locs.GetByHost(host); err != nil {
		if data, err = generateRedirect(locs[0].Vnode, reqURI); err == nil {
			code = statusCodeRedirect
		}
		return
	}

	data, _, err = server.fids.GetKey(key)
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

	//
	// TODO: make following redirects a user controllable option
	//

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
			n = server.conf.Replicas
		}

		code, data, err = server.handleGet(resourceID, n, r.RequestURI)

	case http.MethodPost, http.MethodPut:
		// Append a set operation entry to the log
		var b []byte
		if b, err = ioutil.ReadAll(r.Body); err != nil {
			break
		}
		defer r.Body.Close()

		entry := server.fids.NewEntry([]byte(resourceID))
		entry.Data = append([]byte{fidias.OpSet}, b...)
		code = 200

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
			// Redirect to the next location after us.
			var next *hexaring.Location
			if next, err = meta.PeerSet.GetNext(server.conf.Hostname()); err == nil {
				if data, err = generateRedirect(next.Vnode, r.RequestURI); err == nil {
					code = statusCodeRedirect
				}
			} else {
				// If the above fails redirect to the natural key
				if strings.Contains(err.Error(), "host not in set") {
					// Redirect to the natural key holder
					if data, err = generateRedirect(meta.PeerSet[0].Vnode, r.RequestURI); err == nil {
						code = statusCodeRedirect
					}
				}
			}
			//
			// TODO:
			// During high churn you may reach the max redirect limit.  This may need to
			// be addressed
			//
		}

	case http.MethodDelete:
		// Append a delete operation entry to the log
		entry := server.fids.NewEntry([]byte(resourceID))
		entry.Data = []byte{fidias.OpDel}
		code = 200

		var (
			ballot *hexalog.Ballot
			meta   *fidias.ReMeta
		)
		if ballot, meta, err = server.fids.ProposeEntry(entry); err == nil {
			if err = ballot.Wait(); err == nil {
				data = ballot.Future()
			}
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
