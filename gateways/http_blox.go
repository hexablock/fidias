package gateways

import (
	"encoding/hex"
	"io"
	"log"
	"net/http"

	"github.com/hexablock/blox/block"
	"github.com/hexablock/blox/filesystem"
)

func (server *HTTPServer) handlerBlox(w http.ResponseWriter, r *http.Request, resourceID string) {
	var err error

	switch r.Method {
	case http.MethodGet:
		var id []byte
		if id, err = hex.DecodeString(resourceID); err != nil {
			writeJSONResponse(w, 400, map[string]string{}, nil, err)
			return
		}
		var fh *filesystem.BloxFile
		if fh, err = server.fs.Open(id); err != nil {
			if err == block.ErrBlockNotFound {
				writeJSONResponse(w, 404, map[string]string{}, nil, err)
			} else {
				writeJSONResponse(w, 400, map[string]string{}, nil, err)
			}
			return
		}

		_, err = io.Copy(w, fh)
		fh.Close()

	case http.MethodPost:
		var fh *filesystem.BloxFile
		if fh, err = server.fs.Create(); err != nil {
			writeJSONResponse(w, 400, map[string]string{}, nil, err)
			return
		}

		_, err = io.Copy(fh, r.Body)
		defer r.Body.Close()
		if err != nil {
			writeJSONResponse(w, 400, map[string]string{}, nil, err)
		} else {
			err = fh.Close()
			data := fh.Sys()
			writeJSONResponse(w, 200, map[string]string{}, data, err)
		}

	default:
		w.WriteHeader(405)
		return
	}

	if err != nil {
		log.Printf("[ERROR] FS operation failed: %v", err)
	}

}
