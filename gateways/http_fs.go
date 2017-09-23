package gateways

import (
	"fmt"
	"io"
	"net/http"

	"github.com/hexablock/blox/block"
)

func (server *HTTPServer) handleFS(w http.ResponseWriter, r *http.Request, resourceID string) {
	var err error

	switch r.Method {
	case http.MethodGet:
		err = server.handlerFSGet(w, resourceID)

	case http.MethodPost:
		server.handlerFSPost(w, r, resourceID)
		return

	default:
		w.WriteHeader(405)
		return
	}

	if err == nil {
		return
	}

	var (
		headers map[string]string
		code    int
		data    interface{}
	)

	// TODO:
	// if strings.Contains(err.Error(), "host not in set") {
	// 	code, headers, data, err = buildRedirect(meta.PeerSet, r.URL.RequestURI())
	// }

	writeJSONResponse(w, code, headers, data, err)
}

func (server *HTTPServer) handlerFSGet(w http.ResponseWriter, resourceID string) error {
	fs := server.fids.FileSystem()

	fh, err := fs.Open(resourceID)
	if err != nil {
		var headers map[string]string
		if err == block.ErrBlockNotFound {
			writeJSONResponse(w, 404, headers, nil, err)
		} else {
			writeJSONResponse(w, 400, headers, nil, err)
		}
		return err
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", fh.Size()))
	w.Header().Set(headerBlockSize, fmt.Sprintf("%d", fh.BlockSize()))

	_, err = io.Copy(w, fh)
	fh.Close()

	return err
}

func (server *HTTPServer) handlerFSPost(w http.ResponseWriter, r *http.Request, resourceID string) error {

	headers := map[string]string{}
	fs := server.fids.FileSystem()

	fh, err := fs.Create(resourceID)
	if err != nil {
		writeJSONResponse(w, 400, headers, nil, err)
		return err
	}

	_, err = io.Copy(fh, r.Body)
	defer r.Body.Close()

	if err != nil {
		writeJSONResponse(w, 400, headers, nil, err)
		return err
	}

	// Final response after upload completes assuming there are no errors
	code := 201

	if err = fh.Close(); err == block.ErrBlockExists {
		err = nil
		// The server has fulfilled a request for the resource, and the response is a
		// representation of the result of one or more instance-manipulations applied
		// to the current instance
		code = 226
	}

	data := fh.Sys()
	rt := fh.Runtime()
	headers[headerRuntime] = fmt.Sprintf("%v", rt)
	headers[headerBlockSize] = fmt.Sprintf("%d", fh.BlockSize())
	writeJSONResponse(w, code, headers, data, err)

	return err
}
