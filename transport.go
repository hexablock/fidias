package fidias

import (
	"context"
)

type localKVTransport struct {
	host   string
	kv     KVStore
	remote KVTransport
}

func newLocalKVTransport(host string, remote KVTransport) *localKVTransport {
	return &localKVTransport{
		host:   host,
		remote: remote,
	}
}

func (trans *localKVTransport) GetKey(ctx context.Context, host string, key []byte) (*KVPair, error) {
	if trans.host == host {
		return trans.kv.Get(key)
	}
	return trans.remote.GetKey(ctx, host, key)
}

func (trans *localKVTransport) ListDir(ctx context.Context, host string, dir []byte) ([]*KVPair, error) {
	if trans.host == host {
		out := make([]*KVPair, 0)
		trans.kv.Iter(dir, false, func(kv *KVPair) bool {
			out = append(out, kv)
			return true
		})
		return out, nil
	}

	return trans.remote.ListDir(ctx, host, dir)
}

func (trans *localKVTransport) Register(kv KVStore) {
	// set internal
	trans.kv = kv
	// register with network transport
	trans.remote.Register(kv)
}
