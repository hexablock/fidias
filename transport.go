package fidias

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexatype"
)

// KeyValueTransport implements a transport for key-value operations
type KeyValueTransport interface {
	GetKey(ctx context.Context, host string, key []byte) (*KeyValuePair, error)
}

type FileSystemTransport interface {
	GetPath(ctx context.Context, host string, name string) (*VersionedFile, error)
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
	remote KeyValueTransport
}

func (trans *localKVTransport) GetKey(ctx context.Context, host string, key []byte) (*KeyValuePair, error) {
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

type localFileSystemTransport struct {
	host   string
	local  VersionedFileStore
	remote FileSystemTransport
}

func (trans *localFileSystemTransport) GetPath(ctx context.Context, host string, name string) (*VersionedFile, error) {
	if trans.host == host {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("GetPath context cancelled")
		default:
			return trans.local.GetPath(name)
		}
	}
	return trans.remote.GetPath(ctx, host, name)
}
