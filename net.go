package fidias

import (
	"io"
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/hexablock/blox/device"
	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog/store"
	"github.com/hexablock/hexatype"
)

// LocalStore implements all local calls needed by the network transport
type LocalStore interface {
	KeyValueStore
	VersionedFileStore
}

type streamBase struct {
	o    *rpcOutConn // connection to return
	pool *outPool    // pool to return connection to
}

// Recycle recycles the stream returning the conn back to the pool
func (rs *streamBase) Recycle() {
	rs.pool.returnConn(rs.o)
}

// RelocateStream is a stream to handle relocating of keys between nodes.
type RelocateStream struct {
	*streamBase
	FidiasRPC_RelocateRPCClient // grp stream client

}

type RelocateBlocksStream struct {
	*streamBase
	FidiasRPC_RelocateBlocksRPCClient // grp stream client
}

// NetTransport implements a network transport needed for fidias
type NetTransport struct {
	local LocalStore

	idxs    store.IndexStore // hexalog index store
	journal device.Journal   // BlockDevice journal

	replicas int
	hasher   hexatype.Hasher
	// Incoming relocation requests. i.e. keys this node needs to take over.
	fetCh chan<- *relocateReq
	// Incoming block relocation requests
	fetBlks chan<- *relocateReq

	pool     *outPool
	shutdown int32
}

// NewNetTransport instantiates a new network transport using the given key-value store.
func NewNetTransport(localStore LocalStore, idx store.IndexStore, journal device.Journal, reapInterval, maxIdle time.Duration, replicas int, hasher hexatype.Hasher) *NetTransport {
	return &NetTransport{
		local:    localStore,
		idxs:     idx,
		journal:  journal,
		replicas: replicas,
		hasher:   hasher,
		pool:     newOutPool(maxIdle, reapInterval),
	}
}

// Register registers a write channel used for submitting reloc. requests for keylogs and blocks.
func (trans *NetTransport) Register(fetLogCh, fetBlkCh chan<- *relocateReq) {
	trans.fetCh = fetLogCh
	trans.fetBlks = fetBlkCh
}

// GetKey retrieves a key from a remote host
func (trans *NetTransport) GetKey(ctx context.Context, host string, key []byte) (*KeyValuePair, error) {
	conn, err := trans.pool.getConn(host)
	if err != nil {
		return nil, err
	}

	kvp, err := conn.client.GetKeyRPC(ctx, &KeyValuePair{Key: key})
	if err != nil {
		err = hexatype.ParseGRPCError(err)
	}
	trans.pool.returnConn(conn)

	return kvp, err
}

func (trans *NetTransport) GetPath(ctx context.Context, host string, name string) (*VersionedFile, error) {
	conn, err := trans.pool.getConn(host)
	if err != nil {
		return nil, err
	}
	defer trans.pool.returnConn(conn)

	req := &PathRPC{Name: name}
	resp, err := conn.client.GetPathRPC(ctx, req)
	if err != nil {
		return nil, hexatype.ParseGRPCError(err)
	}
	// New from remote, though we may have one locally as well
	verf := NewVersionedFile(name)
	verf.entry = resp.Entry
	for _, ver := range resp.Versions {
		if err = verf.AddVersion(ver); err != nil {
			break
		}
	}

	return verf, err
}

// GetKeyRPC serves a GetKey request
func (trans *NetTransport) GetKeyRPC(ctx context.Context, in *KeyValuePair) (*KeyValuePair, error) {
	return trans.local.GetKey(in.Key)
}

// GetPathRPC serves a GetPath request
func (trans *NetTransport) GetPathRPC(ctx context.Context, in *PathRPC) (*PathRPC, error) {
	// We don't send the name for efficiency i.e resp.Name = verfile.name, as the
	// requestor already knows the name and can populate it there rather than
	// server sending it again over the wire
	resp := &PathRPC{}

	verfile, err := trans.local.GetPath(in.Name)
	if err != nil {
		return resp, err
	}

	resp.Entry = verfile.entry
	resp.Versions = make([]*FileVersion, len(verfile.versions))
	var i int
	for _, ver := range verfile.versions {
		resp.Versions[i] = ver
		i++
	}

	return resp, nil
}

// GetRelocateStream gets a stream to send relocation keys
func (trans *NetTransport) GetRelocateStream(local, remote *chord.Vnode) (*RelocateStream, error) {
	conn, err := trans.pool.getConn(remote.Host)
	if err != nil {
		return nil, err
	}

	stream, err := conn.client.RelocateRPC(context.Background())
	if err != nil {
		return nil, hexatype.ParseGRPCError(err)
	}

	preamble := &chord.VnodePair{Self: local, Target: remote}
	if err = stream.SendMsg(preamble); err != nil {
		trans.pool.returnConn(conn)
		return nil, hexatype.ParseGRPCError(err)
	}

	return &RelocateStream{
		streamBase:                  &streamBase{o: conn, pool: trans.pool},
		FidiasRPC_RelocateRPCClient: stream,
	}, nil
}

// RelocateRPC serves a GetRelocateStream request stream.  It initiates the process to
// start taking over the sent keys.
func (trans *NetTransport) RelocateRPC(stream FidiasRPC_RelocateRPCServer) error {

	var preamble chord.VnodePair
	if err := stream.RecvMsg(&preamble); err != nil {
		return err
	}

	for {
		keyLoc, err := stream.Recv()
		//	Exit loop on error
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// Create key if it does not exist
		ki, err := trans.idxs.MarkKey(keyLoc.Key, keyLoc.Marker)
		if err != nil {
			log.Printf("[ERROR] Failed to mark key=%s error='%v'", keyLoc.Key, err)
			continue
		}

		// Only continue relocating the key if the marker was set.  If the marker was not set
		// it means we already have the marker entry and nothing needs to be done.
		if ki.Marker() != nil {
			trans.fetCh <- &relocateReq{keyloc: keyLoc, mems: &preamble}
		}
	} // end loop

}

// GetRelocateBlocksStream gets a stream to send relocation keys
func (trans *NetTransport) GetRelocateBlocksStream(local, remote *chord.Vnode) (*RelocateBlocksStream, error) {
	conn, err := trans.pool.getConn(remote.Host)
	if err != nil {
		return nil, err
	}

	stream, err := conn.client.RelocateBlocksRPC(context.Background())
	if err != nil {
		return nil, hexatype.ParseGRPCError(err)
	}

	preamble := &chord.VnodePair{Self: local, Target: remote}
	if err = stream.SendMsg(preamble); err != nil {
		trans.pool.returnConn(conn)
		return nil, hexatype.ParseGRPCError(err)
	}

	return &RelocateBlocksStream{
		streamBase:                        &streamBase{o: conn, pool: trans.pool},
		FidiasRPC_RelocateBlocksRPCClient: stream,
	}, nil
}

// RelocateBlocksRPC serves a GetRelocateStream request stream.  It initiates the process to
// start taking over the sent keys.
func (trans *NetTransport) RelocateBlocksRPC(stream FidiasRPC_RelocateBlocksRPCServer) error {

	var preamble chord.VnodePair
	if err := stream.RecvMsg(&preamble); err != nil {
		return err
	}

	for {
		keyLoc, err := stream.Recv()
		//	Exit loop on error
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		trans.fetBlks <- &relocateReq{keyloc: keyLoc, mems: &preamble}

	} // end loop

}

// Shutdown signals the transport to be shutdown.  After shutdown no new connections can
// be
func (trans *NetTransport) Shutdown() {
	trans.pool.shutdown()
}
