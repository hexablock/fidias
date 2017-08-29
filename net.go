package fidias

import (
	"io"
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog/store"
	"github.com/hexablock/hexatype"
)

// RelocateStream is a stream to handle relocating of keys between nodes.
type RelocateStream struct {
	FidiasRPC_RelocateRPCClient             // grp stream client
	o                           *rpcOutConn // connection to return
	pool                        *outPool    // pool to return connection to
}

// Recycle recycles the stream returning the conn back to the pool
func (rs *RelocateStream) Recycle() {
	rs.pool.returnConn(rs.o)
}

// NetTransport implements a network transport needed for fidias
type NetTransport struct {
	kvs  KeyValueStore
	idxs store.IndexStore

	replicas int
	hasher   hexatype.Hasher
	fetCh    chan<- *relocateReq

	pool     *outPool
	shutdown int32
}

// NewNetTransport instantiates a new network transport using the given key-value store.
func NewNetTransport(kvs KeyValueStore, idx store.IndexStore, reapInterval, maxIdle time.Duration, replicas int, hasher hexatype.Hasher) *NetTransport {
	return &NetTransport{
		kvs:      kvs,
		idxs:     idx,
		replicas: replicas,
		hasher:   hasher,
		pool:     newOutPool(maxIdle, reapInterval),
	}
}

// Register registers a write channel used for submitting reloc. requests
func (trans *NetTransport) Register(ch chan<- *relocateReq) {
	trans.fetCh = ch
}

// GetKey retrieves a key from a remote host
func (trans *NetTransport) GetKey(host string, key []byte) (*hexatype.KeyValuePair, error) {
	conn, err := trans.pool.getConn(host)
	if err != nil {
		return nil, err
	}

	kvp, err := conn.client.GetKeyRPC(context.Background(), &hexatype.KeyValuePair{Key: key})
	if err != nil {
		err = hexatype.ParseGRPCError(err)
	}
	trans.pool.returnConn(conn)

	return kvp, err
}

// GetKeyRPC serves a GetKey request
func (trans *NetTransport) GetKeyRPC(ctx context.Context, in *hexatype.KeyValuePair) (*hexatype.KeyValuePair, error) {
	return trans.kvs.Get(in.Key)
}

// GetRelocateStream gets a stream to send rebalance data across
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

	return &RelocateStream{o: conn, FidiasRPC_RelocateRPCClient: stream, pool: trans.pool}, nil
}

// RelocateRPC serves a rebalance request for the ring
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
		ki, err := trans.idxs.UpsertKey(keyLoc.Key, keyLoc.Marker)
		if err != nil {
			log.Printf("[ERROR] Failed to upsert key=%s error='%v'", keyLoc.Key, err)
			continue
		}

		// Only continue relocating the key if the marker was set.  If the marker was not set
		// it means we already have the marker entry and nothing needs to be done.
		if ki.Marker() != nil {
			trans.fetCh <- &relocateReq{keyloc: keyLoc, mems: &preamble}
		}
	} // end loop

}

// Shutdown signals the transport to be shutdown.  After shutdown no new connections can
// be
func (trans *NetTransport) Shutdown() {
	trans.pool.shutdown()
}
