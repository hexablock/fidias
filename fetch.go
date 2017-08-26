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

type fetcher struct {
	conf *Config
	ring *hexaring.Ring
	hlog *hexalog.Hexalog

	idx     store.IndexStore
	entries store.EntryStore

	trans *localTransport

	fetCh chan *relocateReq // channel of keys to fetch
	chkCh chan []byte       // check channel for after a key has been fetched

	stopped chan struct{}
}

func newFetcher(conf *Config, idx store.IndexStore, ent store.EntryStore, hlog *hexalog.Hexalog, trans *localTransport) *fetcher {
	return &fetcher{
		conf:    conf,
		hlog:    hlog,
		idx:     idx,
		entries: ent,
		trans:   trans,
		fetCh:   make(chan *relocateReq, conf.RebalanceBufSize),
		chkCh:   make(chan []byte, conf.RebalanceBufSize),
		stopped: make(chan struct{}, 2),
	}
}

func (fet *fetcher) register(ring *hexaring.Ring) {
	fet.ring = ring
	fet.start()
}

// fetch fetches a keylog from the given vnode.  If the last entry and marker match then
// fetching is performed.
func (fet *fetcher) fetch(vn *chord.Vnode, key, marker []byte) (*hexalog.FutureEntry, error) {
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

	return fet.trans.remote.FetchKeylog(vn.Host, last, nil)
}

func (fet *fetcher) fetchKeys() {
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

func (fet *fetcher) checkKeys() {
	for key := range fet.chkCh {

		locs, err := fet.ring.LookupReplicated(key, fet.conf.Replicas)
		if err != nil {
			log.Printf("[ERROR] Key check failed key=%s error='%v'", key, err)
			continue
		}

		if err = fet.hlog.Heal(key, &hexatype.RequestOptions{PeerSet: locs}); err != nil {
			log.Printf("[ERROR] Heal failed key=%s error='%v'", key, err)
		}

	}
	// signal we have exited the loop
	fet.stopped <- struct{}{}
}

// start listens to the fetch channel handling each request. It fetches the log for a key
// from the remot in the requets.  This is a blocking call
func (fet *fetcher) start() {
	go fet.checkKeys()
	go fet.fetchKeys()
}

// blocking call
func (fet *fetcher) stop() {
	close(fet.fetCh)
	<-fet.stopped // fetch
	<-fet.stopped // check
}
