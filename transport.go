package fidias

import "github.com/hexablock/hexalog"

// KVNetTransport implements a network transport for key-value operations
type KVNetTransport interface {
	GetKey(host string, key []byte) (*KeyValuePair, error)
}

type localTransport struct {
	host string
	// hexalog local and remote
	local  hexalog.LogStore
	remote hexalog.Transport
	// key-value local and remote
	kvlocal  KeyValueFSM
	kvremote KVNetTransport
}

// GetEntry gets a local or remote entry based on host
func (trans *localTransport) GetEntry(host string, key, id []byte) (*hexalog.Entry, error) {
	if trans.host == host {
		return trans.local.GetEntry(key, id)
	}
	return trans.remote.GetEntry(host, key, id, &hexalog.RequestOptions{})
}

// GetKey gets a local or remote key-value pair based on the host
func (trans *localTransport) GetKey(host string, key []byte) (*KeyValuePair, error) {
	if trans.host == host {
		return trans.kvlocal.Get(key)
	}
	return trans.kvremote.GetKey(host, key)
}
