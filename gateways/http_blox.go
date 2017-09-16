package gateways

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/hexablock/blox/block"
	"github.com/hexablock/blox/filesystem"
)

// Reference: https://stackoverflow.com/questions/2419281/content-length-header-versus-chunked-encoding
//
// Use Content-Length, definitely. The server utilization from this will be almost nonexistent and the
// benefit to your users will be large.
//
// For dynamic content, it's also quite simple to add compressed response support (gzip). That requires
// output buffering, which in turn gives you the content length. (not practical with file downloads or
// already compressed content (sound,images)).
//
// Consider also adding support for partial content/byte-range serving - that is, capability to restart
// downloads. See here for a byte-range example (the example is in PHP, but is applicable in any
// language). You need Content-Length when serving partial content.
//
// Of course, those are not silver bullets: for streaming media, it's pointless to use output buffering
// or response size; for large files, output buffering doesn't make sense, but Content-Length and byte
// serving makes a lot of sense (restarting a failed download is possible).
//
// Personally, I serve Content-Length whenever I know it; for file download, checking the filesize is
// insignificant in terms of resources. Result: user has a determinate progress bar (and dynamic pages
// download faster thanks to gzip).
//

func (server *HTTPServer) handlerBlox(w http.ResponseWriter, r *http.Request, resourceID string) {
	var err error

	switch r.Method {
	case http.MethodGet:
		err = server.handlerBloxGet(w, resourceID)
		// var id []byte
		// if id, err = hex.DecodeString(resourceID); err != nil {
		// 	writeJSONResponse(w, 400, map[string]string{}, nil, err)
		// 	return
		// }

		// var (
		// 	fh *filesystem.BloxFile
		// )

		// if fh, err = server.fs.Open(id); err != nil {
		// 	var headers map[string]string
		// 	if err == block.ErrBlockNotFound {
		// 		writeJSONResponse(w, 404, headers, nil, err)
		// 	} else {
		// 		writeJSONResponse(w, 400, headers, nil, err)
		// 	}
		// 	return
		// }
		// w.Header().Set("Content-Length", fmt.Sprintf("%d", fh.Size()))
		// w.Header().Set(headerBlockSize, fmt.Sprintf("%d", fh.BlockSize()))

		// _, err = io.Copy(w, fh)
		// fh.Close()

	case http.MethodPost:
		err = server.handlerBloxPost(w, r)

	default:
		w.WriteHeader(405)
		return
	}

	if err != nil {
		log.Printf("[ERROR] Blox operation failed: %v", err)
	}

}

func (server *HTTPServer) handlerBloxGet(w http.ResponseWriter, resourceID string) error {
	id, err := hex.DecodeString(resourceID)
	if err != nil {
		writeJSONResponse(w, 400, map[string]string{}, nil, err)
		return err
	}

	var fh *filesystem.BloxFile
	if fh, err = server.fs.Open(id); err != nil {
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

func (server *HTTPServer) handlerBloxPost(w http.ResponseWriter, r *http.Request) error {
	headers := map[string]string{}

	fh, err := server.fs.Create()
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
