package fidias

import (
	"context"
	"io"
	"time"

	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

type LocalNodeProvider interface {
	LocalNode() hexatype.Node
}

// NetTransport is the network transport for fidias as a whole
type NetTransport struct {
	localProv LocalNodeProvider

	kv KVStore

	kvs *KVS

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

// LocalNode returns the LocalNode from the remote host
func (trans *NetTransport) LocalNode(host string) (hexatype.Node, error) {
	var n hexatype.Node
	conn, err := trans.pool.getConn(host)
	if err != nil {
		return n, err
	}

	node, err := conn.client.LocalNodeRPC(context.Background(), &Request{})
	trans.pool.returnConn(conn)
	if err == nil {
		n = *node
	}

	return n, err
}

// GetKey retrieves a key from the single host.  It returns an error if not found
func (trans *NetTransport) GetKey(ctx context.Context, host string, key []byte) (*KVPair, error) {
	conn, err := trans.pool.getConn(host)
	if err != nil {
		return nil, err
	}

	kvp, err := conn.client.GetKeyRPC(ctx, &KVPair{Key: key})
	trans.pool.returnConn(conn)

	return kvp, err
}

// ListDir gets the contents of a directory from a single host
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

// SetRPC serves a set request on the cluster.
func (trans *NetTransport) SetRPC(ctx context.Context, req *WriteRequest) (*WriteResponse, error) {
	kv, stats, err := trans.kvs.Set(req.KV, req.Options)
	if err != nil {
		return nil, err
	}

	resp := &WriteResponse{
		KV: kv,
		Stats: &WriteStats{
			BallotTime:   stats.BallotTime.Nanoseconds(),
			ApplyTime:    stats.ApplyTime.Nanoseconds(),
			Participants: stats.Participants,
		},
	}

	return resp, nil
}

// CASetRPC serves a cluster CASet request
func (trans *NetTransport) CASetRPC(ctx context.Context, req *WriteRequest) (*WriteResponse, error) {
	kv, stats, err := trans.kvs.CASet(req.KV, req.KV.Modification, req.Options)
	if err != nil {
		return nil, err
	}

	resp := &WriteResponse{
		KV: kv,
		Stats: &WriteStats{
			BallotTime:   stats.BallotTime.Nanoseconds(),
			ApplyTime:    stats.ApplyTime.Nanoseconds(),
			Participants: stats.Participants,
		},
	}

	return resp, nil
}

// RemoveRPC serves a cluster Remove request
func (trans *NetTransport) RemoveRPC(ctx context.Context, req *WriteRequest) (*WriteResponse, error) {
	stats, err := trans.kvs.Remove(req.KV.Key, req.Options)
	if err != nil {
		return nil, err
	}

	resp := &WriteResponse{
		Stats: &WriteStats{
			BallotTime:   stats.BallotTime.Nanoseconds(),
			ApplyTime:    stats.ApplyTime.Nanoseconds(),
			Participants: stats.Participants,
		},
	}

	return resp, nil
}

// CARemoveRPC serves a cluster CARemove request
func (trans *NetTransport) CARemoveRPC(ctx context.Context, req *WriteRequest) (*WriteResponse, error) {
	stats, err := trans.kvs.CARemove(req.KV.Key, req.KV.Modification, req.Options)
	if err != nil {
		return nil, err
	}

	resp := &WriteResponse{
		Stats: &WriteStats{
			BallotTime:   stats.BallotTime.Nanoseconds(),
			ApplyTime:    stats.ApplyTime.Nanoseconds(),
			Participants: stats.Participants,
		},
	}

	return resp, nil
}

// GetKeyRPC serves a get key request performing a local lookup
func (trans *NetTransport) GetKeyRPC(ctx context.Context, in *KVPair) (*KVPair, error) {
	log.Printf("[DEBUG] NetTransport.GetKeyRPC key=%s", in.Key)
	return trans.kv.Get(in.Key)
}

// ListDirRPC serves a list dir request from the local store.  It streams all
// kv's for a given dir
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

func (trans *NetTransport) LocalNodeRPC(ctx context.Context, req *Request) (*hexatype.Node, error) {
	node := trans.localProv.LocalNode()
	return &node, nil
}

// Shutdown shuts the outbound connection pool
func (trans *NetTransport) Shutdown() {
	trans.pool.shutdown()
}
