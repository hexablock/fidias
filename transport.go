package fidias

import "github.com/hexablock/hexalog"

type localTransport struct {
	host   string
	local  hexalog.LogStore
	remote hexalog.Transport
}

func (trans *localTransport) GetEntry(host string, key, id []byte) (*hexalog.Entry, error) {
	if trans.host == host {
		return trans.local.GetEntry(key, id)
	}
	return trans.remote.GetEntry(host, key, id, &hexalog.RequestOptions{})
}
