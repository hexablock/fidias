package gateways

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/hexablock/fidias"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
)

// handleHexalog serves http requests to get a keylog and add an entry to the keylog by key
func (server *HTTPServer) handleHexalog(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {
	headers = map[string]string{}

	if resourceID == "" {
		code = 404
		return
	}

	var (
		host = server.conf.Hostname()
		locs hexaring.LocationSet
	)

	if locs, err = server.ring.LookupReplicated([]byte(resourceID), server.conf.Replicas); err != nil {
		return
	}

	// Check if host is part of the location set otherwise re-direct to the natural vnode
	//var loc *hexaring.Location
	if _, err = locs.GetByHost(host); err != nil {
		if strings.Contains(err.Error(), "host not in set") {
			code, headers, data, err = checkHostNotInSetErrorOrRedirect(err, locs, r.RequestURI)
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		code = 200
		keid := strings.Split(resourceID, "/")

		if len(keid) == 2 {
			// Get a secific entry for a key
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

	default:
		code = 405
	}

	return
}
