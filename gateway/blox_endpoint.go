package gateway

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"github.com/hexablock/blox"
	"github.com/hexablock/blox/block"
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

func (server *HTTPServer) handleBlox(w http.ResponseWriter, r *http.Request, resourceID string) {
	var err error

	switch r.Method {
	case http.MethodGet:
		err = server.handlerBloxGet(w, resourceID)

	case http.MethodPost:
		err = server.handlerBloxPost(w, r)

	default:
		w.WriteHeader(405)
		return
	}

	// catch all
	if err != nil {
		writeJSONResponse(w, 400, map[string]string{}, nil, err)
		log.Printf("[ERROR] Blox operation failed: %v", err)
	}

}

func (server *HTTPServer) handlerBloxGet(w http.ResponseWriter, resourceID string) error {
	id, err := hex.DecodeString(resourceID)
	if err != nil {
		return err
	}

	asm := blox.NewAssembler(server.Device, 3)
	idx, err := asm.SetRoot(id)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", idx.FileSize()))
	w.Header().Set(headerBlockSize, fmt.Sprintf("%d", idx.BlockSize()))
	w.Header().Set(headerBlockReadTime, fmt.Sprintf("%v", asm.Runtime()))
	w.Header().Set(headerBlockCount, fmt.Sprintf("%d", idx.BlockCount()))

	//  Cannot send an error
	if err = asm.Assemble(w); err != nil {
		log.Println("[ERROR]", err)
	}

	return nil
}

func (server *HTTPServer) handlerBloxPost(w http.ResponseWriter, r *http.Request) error {
	headers := map[string]string{}

	sharder := blox.NewStreamSharder(server.Device, 3)
	err := sharder.Shard(r.Body)
	if err != nil {
		return err
	}

	// Final response after upload completes assuming there are no errors
	code := 201

	headers[headerBlockWriteTime] = fmt.Sprintf("%v", sharder.Runtime())

	data := sharder.IndexBlock()
	if _, err = server.Device.SetBlock(data); err != nil {

		if err == block.ErrBlockExists {
			// The server has fulfilled a request for the resource, and the response
			// is a representation of the result of one or more instance
			// manipulations applied to the current instance
			code = 226
			err = nil
		}

	}

	writeJSONResponse(w, code, headers, data, err)
	return nil
}
