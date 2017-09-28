package fidias

import (
	"fmt"
	"time"

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
	host   string
	hexlog *hexalog.Hexalog
	store  *hexalog.LogStore
	remote *hexalog.NetTransport
}

func (trans *localHexalogTransport) NewEntry(host string, key []byte) (*hexatype.Entry, error) {
	if trans.host == host {
		return trans.hexlog.New(key), nil
	}
	return trans.remote.NewEntry(host, key, &hexatype.RequestOptions{})
}

func (trans *localHexalogTransport) ProposeEntry(host string, entry *hexatype.Entry, opts *hexatype.RequestOptions) error {
	if trans.host == host {
		ballot, err := trans.hexlog.Propose(entry, opts)
		if err != nil {
			return err
		}
		if !opts.WaitBallot {
			return nil
		}
		if err = ballot.Wait(); err != nil {
			return err
		}
		if opts.WaitApply {
			fut := ballot.Future()
			_, err = fut.Wait(time.Duration(opts.WaitApplyTimeout) * time.Millisecond)
		}

		return err
	}

	ctx := context.Background()
	// No remote ballot
	return trans.remote.ProposeEntry(ctx, host, entry, opts)
}

// GetEntry gets a local or remote entry based on host
func (trans *localHexalogTransport) GetEntry(host string, key, id []byte) (*hexatype.Entry, error) {
	if trans.host == host {
		return trans.store.GetEntry(key, id)
	}
	return trans.remote.GetEntry(host, key, id, &hexatype.RequestOptions{})
}

func (trans *localHexalogTransport) LastEntry(host string, key []byte) (*hexatype.Entry, error) {
	if trans.host == host {
		return trans.store.LastEntry(key), nil
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
