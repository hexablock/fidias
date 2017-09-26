package gateways

import (
	"fmt"
	"io"
	"net/http"

	"github.com/hexablock/blox/block"
	"github.com/hexablock/fidias"
)

func (server *HTTPServer) handleFS(w http.ResponseWriter, r *http.Request, resourceID string) {
	var (
		code    int
		headers map[string]string
		data    interface{}
		err     error
	)

	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()

		if _, ok := q["stat"]; ok {
			code, data, err = server.handleFSStat(w, resourceID)
		} else if _, ok := q["versions"]; ok {
			code, headers, data, err = server.handleFSVersions(w, r, resourceID)
		} else {
			// We return here if there is no error as the handler has written everything
			// needed. It fall through to the write below only if there is an error
			if code, err = server.handleFSGet(w, resourceID); err == nil {
				return
			}
		}

	case http.MethodPost:
		code, headers, data, err = server.handleFSPost(w, r, resourceID)

	default:
		code = 405
	}

	//if strings.Contains(err.Error(), "host not in set") {
	// TODO:
	// 	code, headers, data, err = buildRedirect(meta.PeerSet, r.URL.RequestURI())
	//}

	writeJSONResponse(w, code, headers, data, err)
}

func (server *HTTPServer) handleFSGet(w http.ResponseWriter, resourceID string) (int, error) {
	fs := server.fids.FileSystem()

	code := 200

	fh, err := fs.Open(resourceID)
	if err != nil {
		if err == block.ErrBlockNotFound {
			code = 404
		}
		return code, err
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", fh.Size()))
	w.Header().Set(headerBlockSize, fmt.Sprintf("%d", fh.BlockSize()))

	_, err = io.Copy(w, fh)
	fh.Close()

	return code, err
}

func (server *HTTPServer) handleFSStat(w http.ResponseWriter, resourceID string) (int, interface{}, error) {
	fs := server.fids.FileSystem()

	code := 200

	fh, err := fs.Stat(resourceID)
	if err != nil {
		if err == block.ErrBlockNotFound {
			code = 404
		}
		return code, nil, err
	}

	blk := fh.Sys()
	var data interface{}
	if fh.IsDir() {
		data = blk.(*block.TreeBlock)
	} else {
		data = blk.(*block.IndexBlock)
	}
	return code, data, nil
}

func (server *HTTPServer) handleFSVersions(w http.ResponseWriter, r *http.Request, resourceID string) (int, map[string]string, interface{}, error) {

	fs := server.fids.FileSystem()

	code := 200

	stat, err := fs.Stat(resourceID)
	if err != nil {
		if err == block.ErrBlockNotFound {
			code = 404
		}
		return code, nil, nil, err
	}

	fh := stat.(*fidias.File)
	data := fh.Versions()
	return code, nil, data, nil
}

func (server *HTTPServer) handleFSPost(w http.ResponseWriter, r *http.Request, resourceID string) (int, map[string]string, interface{}, error) {

	fs := server.fids.FileSystem()

	fh, err := fs.Create(resourceID)
	if err != nil {

		if err.Error() == "file exists" {
			// 304 Not Modified
			return 304, nil, nil, nil
		}

		return 400, nil, nil, err
	}

	_, err = io.Copy(fh, r.Body)
	defer r.Body.Close()

	if err != nil {
		return 400, nil, nil, err
	}

	// Final response after upload completes assuming there are no errors
	code := 201
	headers := map[string]string{
		headerBlockWriteTime: fmt.Sprintf("%v", fh.Runtime()),
		headerBlockSize:      fmt.Sprintf("%d", fh.BlockSize()),
	}

	if err = fh.Close(); err == block.ErrBlockExists {
		err = nil
		// The server has fulfilled a request for the resource, and the response is a
		// representation of the result of one or more instance-manipulations applied
		// to the current instance
		code = 226
	}

	return code, headers, fh.Sys(), err
}
