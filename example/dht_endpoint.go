package main

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hexablock/go-kelips"
)

func (server *httpServer) handleDHT(w http.ResponseWriter, r *http.Request, resource string) {

	var (
		err error
	)

	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		id, er := hex.DecodeString(resource)
		if er != nil {
			id = []byte(resource)
		}

		nodes, er := server.dht.Lookup(id)
		if er != nil {
			err = er
			break
		}

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
		err = server.dht.Insert([]byte(hpath[0]), tuple)

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
