package fidias

import (
	"context"
	"fmt"

	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexatype"
)

// KVNetTransport implements a transport for key-value operations
type KVTransport interface {
	GetKey(ctx context.Context, host string, key []byte) (*hexatype.KeyValuePair, error)
}

type localHexalogTransport struct {
	host     string
	logstore *hexalog.LogStore
	remote   *hexalog.NetTransport
}

// GetEntry gets a local or remote entry based on host
func (trans *localHexalogTransport) GetEntry(host string, key, id []byte) (*hexatype.Entry, error) {
	if trans.host == host {
		return trans.logstore.GetEntry(key, id)
	}
	return trans.remote.GetEntry(host, key, id, &hexatype.RequestOptions{})
}

func (trans *localHexalogTransport) LastEntry(host string, key []byte) (*hexatype.Entry, error) {
	if trans.host == host {
		return trans.logstore.LastEntry(key), nil
	}
	return trans.remote.LastEntry(host, key, &hexatype.RequestOptions{})
}

type localKVTransport struct {
	host   string
	local  KeyValueStore
	remote KVTransport
}

func (trans *localKVTransport) GetKey(ctx context.Context, host string, key []byte) (*hexatype.KeyValuePair, error) {
	if trans.host == host {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("GetKey context cancelled")
		default:
			return trans.local.GetKey(key)
		}
	}
	return trans.remote.GetKey(ctx, host, key)
}

// type localTransport struct {
// 	host string
// 	// key-value local
// 	kvlocal KeyValueFSM
// 	// fidias transport as a whole. it contains the required key-value calls as well
// 	trans *NetTransport
// }

// GetKey gets a local or remote key-value pair based on the host.  It also takes a context
// for cancellation of requests
// func (trans *localTransport) GetKey(ctx context.Context, host string, key []byte) (*hexatype.KeyValuePair, error) {
// 	if trans.host == host {
// 		select {
// 		case <-ctx.Done():
// 			return nil, fmt.Errorf("GetKey context cancelled")
// 		default:
// 			return trans.kvlocal.Get(key)
// 		}
// 		//return trans.kvlocal.Get(key)
// 	}
// 	return trans.trans.GetKey(ctx, host, key)
// }
