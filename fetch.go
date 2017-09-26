package fidias

import (
	"bytes"

	"github.com/hexablock/blox"
	"github.com/hexablock/blox/block"
	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexalog/store"
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
	dht DHT
	// Used specifically to submit heal request
	heal Healer
	// Hexalog stores
	idx     store.IndexStore
	entries store.EntryStore

	// block device transport to handle local and remote
	blks blox.Transport

	trans FetcherTransport

	replicas int
	hasher   hexatype.Hasher

	fetCh chan *relocateReq // channel containing of keys to fetch
	chkCh chan []byte       // channel to check key after a fetch is complete
	blkCh chan *relocateReq // channel contianing blocks to fetch

	stopped chan struct{}
}

func NewFetcher(idx store.IndexStore, ent store.EntryStore, replicas, bufSize int, hasher hexatype.Hasher) *Fetcher {
	return &Fetcher{
		idx:      idx,
		entries:  ent,
		replicas: replicas,
		hasher:   hasher,
		fetCh:    make(chan *relocateReq, bufSize),
		chkCh:    make(chan []byte, bufSize),
		blkCh:    make(chan *relocateReq, bufSize),
		stopped:  make(chan struct{}, 3),
	}
}

// RegisterDHT registers the DHT to the fetcher and starts the fetch loop.  This
// must be called after the transport and healer interfaces have been registered.
func (fet *Fetcher) RegisterDHT(dht DHT) {
	fet.dht = dht
	fet.start()
}

func (fet *Fetcher) RegisterTransport(trans FetcherTransport) {
	fet.trans = trans
}

func (fet *Fetcher) RegisterHealer(healer Healer) {
	fet.heal = healer
}

func (fet *Fetcher) RegisterBlockTransport(blks blox.Transport) {
	fet.blks = blks
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

	// Close the check channel
	close(fet.chkCh)
	// signal we have exited the loop
	fet.stopped <- struct{}{}
}

func (fet *Fetcher) checkKeys() {
	for key := range fet.chkCh {

		locs, err := fet.dht.LookupReplicated(key, fet.replicas)
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

func (fet *Fetcher) fetchBlocks() {
	for rr := range fet.blkCh {
		id := rr.keyloc.Key

		// Get local blox address
		ll, _ := rr.mems.Target.Metadata()["blox"]
		local := string(ll)
		// Skip if we have the block
		_, err := fet.blks.GetBlock(local, id)
		if err == nil {
			continue
		}

		// Marker contains the type and size at the bare minimum
		m := rr.keyloc.Marker
		typ := block.BlockType(m[0])

		// Remote host to get block from
		bb, _ := rr.mems.Self.Metadata()["blox"]
		remote := string(bb)

		var blk block.Block
		switch typ {
		case block.BlockTypeData:
			// Actually fetch
			blk, err = fet.blks.GetBlock(remote, id)

		case block.BlockTypeIndex:
			// Inline block
			index := block.NewIndexBlock(nil, fet.hasher)
			if err = index.UnmarshalBinary(m); err == nil {
				index.Hash()
				blk = index
			}

		case block.BlockTypeTree:
			// Inline block
			tree := block.NewTreeBlock(nil, fet.hasher)
			if err = tree.UnmarshalBinary(m); err == nil {
				tree.Hash()
				blk = tree
			}

		default:
			err = block.ErrInvalidBlockType
			continue

		}

		if err != nil {
			log.Printf("[ERROR] Fetcher failed get block id=%x type=%s error='%v'", id, typ, err)
			continue
		}

		// Set the block locally
		if _, err = fet.blks.SetBlock(local, blk); err != nil && err != block.ErrBlockExists {
			log.Printf("[ERROR] Fetcher failed set block id=%x error='%v'", id, err)
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
	go fet.fetchBlocks()
}

// blocking call
func (fet *Fetcher) stop() {
	// close keylog fetcher which will close the key check channel as well
	close(fet.fetCh)
	// close block fetcher channel
	close(fet.blkCh)
	// Wait for all 3 go-routines to complete
	<-fet.stopped // fetcher
	<-fet.stopped // key checker
	<-fet.stopped // block fetcher
}
