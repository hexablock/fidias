package fidias

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexatype"
)

// KVTransport implements a transport for key-value operations
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
