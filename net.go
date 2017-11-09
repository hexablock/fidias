package fidias

import (
	"context"
	"io"
	"time"

	kelips "github.com/hexablock/go-kelips"
	"github.com/hexablock/log"
)

// NetTransport is the network transport for fidias as a whole
type NetTransport struct {
	klp *kelips.Kelips

	kv   KVStore
	pool *outPool
}

// NewNetTransport inits the outbound connections pool and returns an instance
// of NetTransport
func NewNetTransport(reapInterval, maxIdle time.Duration) *NetTransport {
	trans := &NetTransport{
		pool: newOutPool(maxIdle, reapInterval),
	}

	// Start outbound connection pool
	go trans.pool.reapOld()

	return trans
}

// Register registers a KVStore the transport will use to serve requests
func (trans *NetTransport) Register(kvs KVStore) {
	trans.kv = kvs
}

// GetKey retrieves a key from a remote host
func (trans *NetTransport) GetKey(ctx context.Context, host string, key []byte) (*KVPair, error) {
	conn, err := trans.pool.getConn(host)
	if err != nil {
		return nil, err
	}

	kvp, err := conn.client.GetKeyRPC(ctx, &KVPair{Key: key})
	trans.pool.returnConn(conn)

	return kvp, err
}

// ListDir gets the contents of a directory from the host
func (trans *NetTransport) ListDir(ctx context.Context, host string, dir []byte) ([]*KVPair, error) {
	conn, err := trans.pool.getConn(host)
	if err != nil {
		return nil, err
	}

	stream, err := conn.client.ListDirRPC(context.Background(), &KVPair{Key: dir})
	defer trans.pool.returnConn(conn)

	out := make([]*KVPair, 0)
	for {
		kvp, er := stream.Recv()
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
		out = append(out, kvp)
	}

	return out, err
}

// GetKeyRPC serves a get key request performing a local lookup
func (trans *NetTransport) GetKeyRPC(ctx context.Context, in *KVPair) (*KVPair, error) {
	log.Printf("[DEBUG] NetTransport.GetKeyRPC key=%s", in.Key)
	return trans.kv.Get(in.Key)
}

// ListDirRPC serves a list dir request.  It sends all the kv's for a given dir
func (trans *NetTransport) ListDirRPC(in *KVPair, stream FidiasRPC_ListDirRPCServer) error {
	var err error
	trans.kv.Iter(in.Key, false, func(kv *KVPair) bool {
		if err = stream.Send(kv); err != nil {
			return false
		}
		return true
	})

	return err
}

// Shutdown shuts the outbound connection pool
func (trans *NetTransport) Shutdown() {
	trans.pool.shutdown()
}
