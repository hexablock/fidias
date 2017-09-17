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
	locator *hexaring.Ring
	// Hexalog kv write operations
	hexlog *Hexalog
	// Transport for kv read operations
	trans *localKVTransport
}

// NewKeyvs inits a new instance of Keyvs.  It takes the hexalog for write ops,
// key-value store and network transport for read ops
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

// RegisterLocator registers the locator interface
func (kv *Keyvs) RegisterLocator(locator *hexaring.Ring) {
	kv.locator = locator
}

// RegisterTransport registers the remote transport to use.
func (kv *Keyvs) RegisterTransport(remote KVTransport) {
	kv.trans.remote = remote
}

// GetKey requests a key from the nodes in the key peerset concurrently and returns the first
// non-errored result.  If the key is not found in any of the locations, a ErrKeyNotFound is
// returned
func (kv *Keyvs) GetKey(key []byte) (kvp *KeyValuePair, opt *hexatype.RequestOptions, err error) {

	locs, err := kv.locator.LookupReplicated(key, kv.hexlog.MinVotes())
	if err != nil {
		return nil, nil, err
	}
	opt = &hexatype.RequestOptions{PeerSet: locs}

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

func (kv *Keyvs) SetKey(basekey, val []byte) (*hexalog.FutureEntry, *hexatype.RequestOptions, error) {
	key := append(kv.ns, basekey...)

	ballot, opt, err := kv.submitLogEntry(key, append([]byte{OpSet}, val...))
	if err != nil {
		return nil, opt, err
	}

	// TODO:

	err = ballot.Wait()
	fut := ballot.Future()
	return fut, opt, err
}

func (kv *Keyvs) RemoveKey(basekey []byte) (*hexalog.FutureEntry, *hexatype.RequestOptions, error) {
	key := append(kv.ns, basekey...)

	ballot, opt, err := kv.submitLogEntry(key, []byte{OpDel})
	if err != nil {
		return nil, opt, err
	}

	// TODO:

	err = ballot.Wait()
	fut := ballot.Future()
	return fut, opt, err
}

func (kv *Keyvs) submitLogEntry(key []byte, data []byte) (*hexalog.Ballot, *hexatype.RequestOptions, error) {

	entry, opts, err := kv.hexlog.NewEntry(key)
	if err != nil {
		return nil, opts, err
	}
	entry.Data = data

	ballot, err := kv.hexlog.ProposeEntry(entry, opts)
	if err != nil {
		return nil, opts, err
	}

	return ballot, opts, nil
}

// GetKey requests a key from the nodes in the key peerset concurrently and returns the first
// non-errored result.  If the key is not found in any of the locations, a ErrKeyNotFound is
// returned
// func (fidias *Fidias) GetKey(key []byte) (kvp *KeyValuePair, meta *ReMeta, err error) {
// 	locs, err := fidias.locator.LookupReplicated(key, fidias.conf.Hexalog.Votes)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	meta = &ReMeta{PeerSet: locs}
//
// 	ll := len(locs)
// 	resp := make(chan *kvitemError, ll)
// 	ctx, cancel := context.WithCancel(context.Background())
//
// 	for _, l := range locs {
//
// 		go func(k []byte, loc *hexaring.Location) {
// 			kvi := &kvitemError{loc: loc}
// 			kvi.kv, kvi.err = fidias.ftrans.GetKey(ctx, loc.Host(), k)
// 			resp <- kvi
//
// 		}(key, l)
//
// 	}
//
// 	defer cancel()
//
// 	for i := 0; i < ll; i++ {
// 		kvi := <-resp
// 		if kvi.err == nil {
// 			meta.Vnode = kvi.loc.Vnode
// 			return
// 		}
// 	}
//
// 	err = hexatype.ErrKeyNotFound
// 	return
// }
