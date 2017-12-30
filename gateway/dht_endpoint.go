package gateway

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hexablock/go-kelips"
)

func (server *HTTPServer) handleDHT(w http.ResponseWriter, r *http.Request, resource string) {
	if resource == "" {
		w.WriteHeader(404)
		return
	}

	var err error

	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		id, er := hex.DecodeString(resource)
		if er != nil {
			id = []byte(resource)
		}

		start := time.Now()
		nodes, er := server.DHT.Lookup(id)
		if er != nil {
			err = er
			break
		}
		end := time.Since(start)

		w.Header().Set(headerRespTime, fmt.Sprintf("%v", end))
		b, er := json.Marshal(nodes)
		if er != nil {
			err = er
			break
		}
		w.Write(b)

	case "POST":
		hpath := strings.Split(resource, "/")
		if len(hpath) != 2 {
			w.WriteHeader(404)
			return
		}

		tuple := kelips.NewTupleHost(hpath[1])
		err = server.DHT.Insert([]byte(hpath[0]), tuple)

	default:
		w.WriteHeader(405)
		w.Write([]byte(`{"error": "Method not allowed"}`))
		return
	}

	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(`{"error": "` + err.Error() + `"}`))
	}

}
