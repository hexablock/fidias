package fidias

import (
	"context"
	"fmt"

	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexatype"
)

// KVNetTransport implements a network transport for key-value operations
type KVNetTransport interface {
	GetKey(host string, key []byte) (*hexatype.KeyValuePair, error)
}

type localTransport struct {
	host string
	// hexalog local
	local *hexalog.LogStore
	// hexalog remote
	remote hexalog.Transport
	// key-value local
	kvlocal KeyValueFSM
	// fidias transport as a whole. it contains the required key-value calls as well
	ftrans *NetTransport
}

// GetEntry gets a local or remote entry based on host
func (trans *localTransport) GetEntry(host string, key, id []byte) (*hexatype.Entry, error) {
	if trans.host == host {
		return trans.local.GetEntry(key, id)
	}
	return trans.remote.GetEntry(host, key, id, &hexatype.RequestOptions{})
}

func (trans *localTransport) LastEntry(host string, key []byte) (*hexatype.Entry, error) {
	if trans.host == host {
		return trans.local.LastEntry(key), nil
	}
	return trans.remote.LastEntry(host, key, &hexatype.RequestOptions{})
}

// GetKey gets a local or remote key-value pair based on the host
func (trans *localTransport) GetKey(ctx context.Context, host string, key []byte) (*hexatype.KeyValuePair, error) {
	if trans.host == host {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("GetKey context cancelled")
		default:
			return trans.kvlocal.Get(key)
		}
		//return trans.kvlocal.Get(key)
	}
	return trans.ftrans.GetKey(ctx, host, key)
}
