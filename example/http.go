package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hexablock/fidias3"
	"github.com/hexablock/hexatype"
)

const (
	headerBallotTime     = "Ballot-Time"
	headerBlockSize      = "Block-Size"
	headerBlockWriteTime = "Block-Write-Time"
	headerBlockReadTime  = "Block-Read-Time"
	headerBlockCount     = "Block-Count"
	headerFsmTime        = "Fsm-Time"
	headerGroup          = "Group-Index"
	headerLookupTime     = "Lookup-Time"
	headerNodeHBeat      = "Node-Heartbeats"
	headerNodeRTT        = "Node-Rtt"
	headerNodePriority   = "Node-Priority"
	headerParticipants   = "Participants"
	headerRespTime       = "Response-Time"
	headerRuntime        = "Runtime"
)

var accessControlHeaders = map[string]string{
	"Access-Control-Allow-Origin": "*",
}

type httpServer struct {
	dht fidias.DHT
	dev *fidias.BlockDevice
	kvs *fidias.KVS
}

func (server *httpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqpath := r.URL.Path[1:]
	if reqpath == "" {
		w.WriteHeader(404)
		return
	}

	endpoint, resource := parseDirBase(reqpath)

	switch endpoint {
	case "dht":
		if resource == "" {
			w.WriteHeader(404)
			return
		}
		server.handleDHT(w, r, resource)

	case "blox":
		server.handleBlox(w, r, resource)

	case "kv":
		if resource == "" {
			w.WriteHeader(404)
			return
		}
		server.handleKV(w, r, resource)

	default:
		w.WriteHeader(404)
	}

}

func setWriteHeaderStats(w http.ResponseWriter, stats *fidias.WriteStats) {
	w.Header().Set(headerBallotTime, fmt.Sprintf("%v", stats.BallotTime))
	w.Header().Set(headerFsmTime, fmt.Sprintf("%v", stats.ApplyTime))

	hosts := make([]string, 0, len(stats.Participants))
	for _, p := range stats.Participants {
		hosts = append(hosts, fmt.Sprintf("%s/%x", p.Host, p.ID))
	}
	w.Header().Set(headerParticipants, strings.Join(hosts, ","))
}

// setReadHeader sets common read headers.
func setReadHeader(w http.ResponseWriter, stats *fidias.ReadStats) {
	var nodes string
	for _, n := range stats.Nodes {
		nodes += fmt.Sprintf("%s/%x,", n.Host(), n.ID)
	}
	w.Header().Set(headerParticipants, nodes[:len(nodes)-1])
	w.Header().Set(headerRespTime, stats.RespTime.String())
}

func setNodeGroupHeaders(w http.ResponseWriter, g, p int, node hexatype.Node) {
	w.Header().Set(headerGroup, fmt.Sprintf("%d", g))
	w.Header().Set(headerNodeHBeat, fmt.Sprintf("%d", node.Heartbeats))
	w.Header().Set(headerNodePriority, fmt.Sprintf("%d", p))
}

// writeJSONResponse writes a json response.  It first sets the headers, then the code and
// finally the data.  It manages serializing the data.  It data is a byte slice then it simply
// writes the data without setting any content type headers
func writeJSONResponse(w http.ResponseWriter, code int, headers map[string]string, data interface{}, err error) {

	w.Header().Set("Content-Type", "application/json")

	var (
		b []byte
		c = code
	)

	// Make sure the code is > 400 if it is an error otherwise set it to 400
	if err != nil {
		if code < 400 {
			c = 400
		}
		// Error data
		b = []byte(`{"error":"` + err.Error() + `"}`)

	} else {
		if data != nil {
			// Write byte slice directly
			if t, ok := data.([]byte); ok {
				b = t
			} else {
				b, _ = json.Marshal(data)
			}
		}

	}

	// Set headers supplied as input
	if headers != nil {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
	}

	// Set ACL headers after so they are not overwritten by the caller
	for k, v := range accessControlHeaders {
		w.Header().Set(k, v)
	}

	// Write code
	w.WriteHeader(c)
	// Write data
	w.Write(b)
}
