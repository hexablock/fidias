package gateways

import (
	"net/http"

	"github.com/hexablock/hexaring"
)

func (server *HTTPServer) handleLeader(w http.ResponseWriter, r *http.Request, resourceID string) (code int, headers map[string]string, data interface{}, err error) {
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
		n = server.conf.Replicas
	}

	var locs hexaring.LocationSet
	if locs, err = server.ring.LookupReplicated([]byte(resourceID), n); err != nil {
		return
	}

	//var leader *fidias.KeyLeader
	data, err = server.fids.Leader([]byte(resourceID), locs)

	return
}
