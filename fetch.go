package fidias

import (
	"bytes"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexalog/store"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

type Healer interface {
	Heal(key []byte, opts *hexatype.RequestOptions) error
}

type FetcherTransport interface {
	FetchKeylog(host string, entry *hexatype.Entry, opts *hexatype.RequestOptions) (*hexalog.FutureEntry, error)
}

type Fetcher struct {
	locator *hexaring.Ring

	// Used specifically to submit heal request
	heal Healer

	idx     store.IndexStore
	entries store.EntryStore
	// Hexalog network transport
	//trans hexalog.Transport

	trans FetcherTransport

	replicas int

	fetCh chan *relocateReq // channel of keys to fetch
	chkCh chan []byte       // check channel for after a key has been fetched

	stopped chan struct{}
}

func NewFetcher(idx store.IndexStore, ent store.EntryStore, replicas, bufSize int) *Fetcher {
	return &Fetcher{
		idx:      idx,
		entries:  ent,
		replicas: replicas,
		fetCh:    make(chan *relocateReq, bufSize),
		chkCh:    make(chan []byte, bufSize),
		stopped:  make(chan struct{}, 2),
	}
}

// RegisterLocator registers the locator to the fetcher and starts the fetch loop.  This
// must be called after the transport and healer interfaces have been registered.
func (fet *Fetcher) RegisterLocator(locator *hexaring.Ring) {
	fet.locator = locator
	fet.start()
}

func (fet *Fetcher) RegisterTransport(trans FetcherTransport) {
	fet.trans = trans
}

func (fet *Fetcher) RegisterHealer(healer Healer) {
	fet.heal = healer
}

// fetch fetches a keylog from the given vnode.  If the last entry and marker match then
// fetching is not performed.
func (fet *Fetcher) fetch(vn *chord.Vnode, key, marker []byte) (*hexalog.FutureEntry, error) {
	keyidx, err := fet.idx.GetKey(key)
	if err != nil {
		return nil, err
	}

	var (
		last *hexatype.Entry
		lid  = keyidx.Last()
	)
	if lid != nil {
		// Skip if marker and last entry id match
		if bytes.Compare(lid, marker) == 0 {
			return nil, nil
		}
		// Get the last entry
		last, _ = fet.entries.Get(lid)
	}

	if last == nil {
		last = &hexatype.Entry{Key: key}
	}

	return fet.trans.FetchKeylog(vn.Host, last, nil)
}

func (fet *Fetcher) fetchKeys() {
	for req := range fet.fetCh {
		// self is the remote node that has the data.  Key entries will be fetched from this
		// vnode
		src := req.mems.Self
		kl := req.keyloc

		// Fetch the keylog from the remote node
		if _, err := fet.fetch(src, kl.Key, kl.Marker); err != nil {
			log.Printf("[ERROR] Failed to fetch key=%s src=%s/%x error='%v'", kl.Key, src.Host,
				src.Id[:12], err)

		}
		// Send key to be checked
		fet.chkCh <- kl.Key

	}

	// close the check channel
	close(fet.chkCh)
	// signal we have exited the loop
	fet.stopped <- struct{}{}
}

func (fet *Fetcher) checkKeys() {
	for key := range fet.chkCh {

		locs, err := fet.locator.LookupReplicated(key, fet.replicas)
		if err != nil {
			log.Printf("[ERROR] Key check failed key=%s error='%v'", key, err)
			continue
		}

		if err = fet.heal.Heal(key, &hexatype.RequestOptions{PeerSet: locs}); err != nil {
			log.Printf("[ERROR] Heal failed key=%s error='%v'", key, err)
		}

	}
	// signal we have exited the loop
	fet.stopped <- struct{}{}
}

// start listens to the fetch channel handling each request. It fetches the log for a key
// from the remot in the requets.  This is a blocking call
func (fet *Fetcher) start() {
	go fet.checkKeys()
	go fet.fetchKeys()
}

// blocking call
func (fet *Fetcher) stop() {
	close(fet.fetCh)
	<-fet.stopped // fetch
	<-fet.stopped // check
}
