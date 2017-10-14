package fidias

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"github.com/hexablock/hexalog"
)

// Transport contains internal transports to handle local and remote operations
// using a single interface

// KeyValueTransport implements a transport for remote key-value operations
type KeyValueTransport interface {
	GetKey(ctx context.Context, host string, key []byte) (*KeyValuePair, error)
}

// FileSystemTransport implements a transport for remote filesystem operations
type FileSystemTransport interface {
	GetPath(ctx context.Context, host string, name string) (*VersionedFile, error)
}

type localHexalogTransport struct {
	host   string
	hexlog *hexalog.Hexalog
	store  *hexalog.LogStore
	remote *hexalog.NetTransport
}

func (trans *localHexalogTransport) NewEntry(host string, key []byte) (*hexalog.Entry, error) {
	if trans.host == host {
		return trans.hexlog.New(key), nil
	}
	return trans.remote.NewEntry(host, key, &hexalog.RequestOptions{})
}

func (trans *localHexalogTransport) ProposeEntry(host string, entry *hexalog.Entry, opts *hexalog.RequestOptions) error {
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
func (trans *localHexalogTransport) GetEntry(host string, key, id []byte) (*hexalog.Entry, error) {
	if trans.host == host {
		return trans.store.GetEntry(key, id)
	}
	return trans.remote.GetEntry(host, key, id, &hexalog.RequestOptions{})
}

func (trans *localHexalogTransport) LastEntry(host string, key []byte) (*hexalog.Entry, error) {
	if trans.host == host {
		return trans.store.LastEntry(key), nil
	}
	return trans.remote.LastEntry(host, key, &hexalog.RequestOptions{})
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
