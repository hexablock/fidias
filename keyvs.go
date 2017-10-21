package fidias

import (
	"context"

	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
)

type kvitemError struct {
	loc *hexaring.Location
	kv  *KeyValuePair
	err error
}

// Keyvs is a key-value interface that relies on Hexalog and Hexaring to provide
// functions to perform CRUD like operations on keys
type Keyvs struct {
	// namespace
	ns []byte
	// Needed for read requests
	dht DHT
	// Hexalog kv write operations
	hexlog *Hexalog
	// Transport for kv read operations
	trans *localKVTransport
}

// NewKeyvs inits a new instance of Keyvs.  It takes the hexalog for write ops,
// key-value store and network transport for read ops.  namespace is used to
// prefix all keys.
func NewKeyvs(namespace string, hexlog *Hexalog, kvs KeyValueStore) *Keyvs {
	trans := &localKVTransport{
		host:  hexlog.conf.Hostname,
		local: kvs,
	}

	return &Keyvs{
		ns:     []byte(namespace),
		hexlog: hexlog,
		trans:  trans,
	}
}

// RegisterDHT registers the ring to the keyvalue store
func (kv *Keyvs) RegisterDHT(dht DHT) {
	kv.dht = dht
}

// RegisterTransport registers the remote transport to use.
func (kv *Keyvs) RegisterTransport(remote KeyValueTransport) {
	kv.trans.remote = remote
}

// GetKey requests a key from the nodes in the key peerset concurrently and returns the first
// non-errored result.  If the key is not found in any of the locations, a ErrKeyNotFound is
// returned
func (kv *Keyvs) GetKey(key []byte) (kvp *KeyValuePair, opt *hexalog.RequestOptions, err error) {
	nskey := append(kv.ns, key...)

	locs, err := kv.dht.LookupReplicated(nskey, kv.hexlog.MinVotes())
	if err != nil {
		return nil, nil, err
	}
	participants := locationsToParticipants(locs)
	opt = &hexalog.RequestOptions{PeerSet: participants}

	ll := len(locs)
	resp := make(chan *kvitemError, ll)
	ctx, cancel := context.WithCancel(context.Background())

	for _, l := range locs {

		go func(k []byte, loc *hexaring.Location) {
			kvi := &kvitemError{loc: loc}
			kvi.kv, kvi.err = kv.trans.GetKey(ctx, loc.Host(), k)
			resp <- kvi

		}(key, l)

	}

	defer cancel()

	for i := 0; i < ll; i++ {
		kvi := <-resp
		if kvi.err == nil {
			//meta.Vnode = kvi.loc.Vnode
			kvp = kvi.kv
			return
		}
	}

	err = hexatype.ErrKeyNotFound
	return
}

// SetKey sets a key to the value
func (kv *Keyvs) SetKey(basekey, val []byte) (*hexalog.Entry, *hexalog.RequestOptions, error) {
	key := append(kv.ns, basekey...)

	return kv.submitLogEntry(key, append([]byte{OpSet}, val...))
}

// RemoveKey removes a key
func (kv *Keyvs) RemoveKey(basekey []byte) (*hexalog.Entry, *hexalog.RequestOptions, error) {
	key := append(kv.ns, basekey...)

	return kv.submitLogEntry(key, []byte{OpDel})
}

// generic function for write operations
func (kv *Keyvs) submitLogEntry(key []byte, data []byte) (*hexalog.Entry, *hexalog.RequestOptions, error) {

	entry, opts, err := kv.hexlog.NewEntry(key)
	if err != nil {
		return nil, opts, err
	}
	entry.Data = data

	opts.WaitBallot = true
	err = kv.hexlog.ProposeEntry(entry, opts)

	return entry, opts, err
}
