package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/hexablock/fidias"
)

func (server *httpServer) handleKV(w http.ResponseWriter, r *http.Request, resource string) {
	var (
		data  interface{}
		err   error
		stats *fidias.WriteStats
		key   = []byte(resource)
	)

	switch r.Method {

	case "GET":
		var (
			rstats *fidias.ReadStats
			kv     *fidias.KVPair
		)

		// Get KVPair
		if kv, rstats, err = server.kvs.Get(key, nil); err != nil {
			break
		}

		// List contents if directory
		if kv.IsDir() {
			data, rstats, err = server.kvs.List(key, &fidias.ReadOptions{})
		} else {
			data = kv
			setNodeGroupHeaders(w, rstats.Group, rstats.Priority, *rstats.Nodes[0])
		}

		setReadHeader(w, rstats)

	case "POST":
		var value []byte
		if value, err = ioutil.ReadAll(r.Body); err == nil {
			wo := fidias.DefaultWriteOptions()
			kv := fidias.NewKVPair([]byte(resource), value)
			data, stats, err = server.kvs.Set(kv, wo)
		}

	case "DELETE":
		wo := fidias.DefaultWriteOptions()
		stats, err = server.kvs.Remove([]byte(resource), wo)

	}

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		return
	}

	if stats != nil {
		setWriteHeaderStats(w, stats)
	}

	b, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	w.Write(b)

}

func parseDirBase(path string) (string, string) {
	var i int
	for j, c := range path {
		if c == '/' {
			i = j
			break
		}
	}
	if i == 0 {
		return path, ""
	}

	return path[:i], path[i+1:]
}
