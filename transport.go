package fidias

import "github.com/hexablock/hexalog"

// KVNetTransport implements a network transport for key-value operations
type KVNetTransport interface {
	GetKey(host string, key []byte) (*KeyValuePair, error)
}

type localTransport struct {
	host string

	local  hexalog.LogStore
	remote hexalog.Transport

	kvlocal  KeyValueFSM
	kvremote KVNetTransport
}

func (trans *localTransport) GetEntry(host string, key, id []byte) (*hexalog.Entry, error) {
	if trans.host == host {
		return trans.local.GetEntry(key, id)
	}
	return trans.remote.GetEntry(host, key, id, &hexalog.RequestOptions{})
}

func (trans *localTransport) GetKey(host string, key []byte) (*KeyValuePair, error) {
	if trans.host == host {
		return trans.kvlocal.Get(key)
	}
	return trans.kvremote.GetKey(host, key)
}
