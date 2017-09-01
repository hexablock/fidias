package gateways

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hexablock/fidias"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
)

func (server *HTTPServer) handleGetKey(resourceID string, n int, reqURI string) (code int, headers map[string]string, data interface{}, err error) {
	var (
		host = server.conf.Hostname()
		key  = []byte(resourceID)
		locs hexaring.LocationSet
	)
	headers = map[string]string{}

	// Find the starting position
	if locs, err = server.ring.LookupReplicated(key, n); err != nil {
		return
	}

	// Check if host is part of the location set otherwise re-direct to the natural vnode
	var loc *hexaring.Location
	if loc, err = locs.GetByHost(host); err != nil {
		if strings.Contains(err.Error(), "host not in set") {
			code, headers, data, err = checkHostNotInSetErrorOrRedirect(err, locs, reqURI)
		}
		return
	}
	headers[headerLocations] = fmt.Sprintf("%s/%x", loc.Vnode.Host, loc.Vnode.Id)

	data, err = server.fsm.Get(key)
	// var meta *fidias.ReMeta
	// data, meta, err = server.fids.GetKey(key)
	if err == nil {
		//headers[headerLocations] = locationSetHeaderVals(meta.PeerSet)
		code = 200
	} else if err == hexatype.ErrKeyNotFound {
		code = 404
	}

	return
}

func (server *HTTPServer) handleWriteKey(resourceID string, op byte, reqData []byte, reqURI string) (code int, headers map[string]string, data interface{}, err error) {
	headers = map[string]string{}

	entry, opts, err := server.fids.NewEntry([]byte(resourceID))
	if err != nil {
		if strings.Contains(err.Error(), "host not in set") {
			code, headers, data, err = checkHostNotInSetErrorOrRedirect(err, opts.PeerSet, reqURI)
		}
		return
	}

	entry.Data = append([]byte{op}, reqData...)
	code = 200

	opts.Retries = 2

	var ballot *hexalog.Ballot
	ballot, err = server.fids.ProposeEntry(entry, opts)
	if err != nil {
		return
	}

	if err = ballot.Wait(); err == nil {
		data = ballot.Future()
		headers[headerLocations] = locationSetHeaderVals(opts.PeerSet)
	}

	// Runtime headers
	headers[headerBallotTime] = fmt.Sprintf("%v", ballot.Runtime())
	return
}

func (server *HTTPServer) handleKeyValue(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {
	if resourceID == "" {
		code = 404
		return
	}

	// Parameters
	var n int
	n, err = parseIntQueryParam(r, "n")
	if err != nil {
		return
	}
	if n == 0 {
		n = server.conf.Hexalog.Votes
	}

	code = 200

	switch r.Method {
	case http.MethodGet:
		code, headers, data, err = server.handleGetKey(resourceID, n, r.RequestURI)

	case http.MethodPost, http.MethodPut:
		// Append a set operation entry to the log
		var b []byte
		if b, err = ioutil.ReadAll(r.Body); err != nil {
			break
		}
		defer r.Body.Close()

		code, headers, data, err = server.handleWriteKey(resourceID, fidias.OpSet, b, r.RequestURI)

	case http.MethodDelete:
		code, headers, data, err = server.handleWriteKey(resourceID, fidias.OpDel, []byte{}, r.RequestURI)

	default:
		code = 405
	}

	return
}
