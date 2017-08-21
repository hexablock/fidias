package fidias

import (
	"bytes"
	"log"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexalog/store"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
)

// RelocatorTransport implements a transport needed by the key rebalancing engine
type RelocatorTransport interface {
	GetRelocateStream(local, remote *chord.Vnode) (*RelocateStream, error)
}

// rebReq contains data for perform a relocation
type relocateReq struct {
	keyloc *KeyLocation
	mems   *chord.VnodePair
}

// Relocator is responsible for moving data as needed when the underlying cluster topology
// changes
type Relocator struct {
	idx     store.IndexStore
	entries store.EntryStore

	keyblks *keyBlockSet // owned vnode blocks

	trans    RelocatorTransport
	logtrans hexalog.Transport

	hasher   hexatype.Hasher
	replicas int

	fetCh   chan *relocateReq // channel of keys to fetch
	stopped chan struct{}     // signal everything has stopped
}

// NewRelocator instantiates a new Relocator
func NewRelocator(kbs *keyBlockSet, idx store.IndexStore, entries store.EntryStore, trans RelocatorTransport,
	logtrans hexalog.Transport, hasher hexatype.Hasher, replicas int, rebBufSize int) *Relocator {

	return &Relocator{
		keyblks:  kbs,
		idx:      idx,
		entries:  entries,
		trans:    trans,
		logtrans: logtrans,
		replicas: replicas,
		hasher:   hasher,
		fetCh:    make(chan *relocateReq, rebBufSize),
		stopped:  make(chan struct{}, 1),
	}

}

// fetch fetches a keylog from the given vnode.  If the last entry and marker match then
// fetching is performed.
func (reb *Relocator) fetch(vn *chord.Vnode, key, marker []byte) (*hexalog.FutureEntry, error) {
	//local:=req.mems.Target
	keyidx, err := reb.idx.GetKey(key)
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
		last, _ = reb.entries.Get(lid)
	}

	if last == nil {
		last = &hexatype.Entry{Key: key}
	}

	log.Printf("[DEBUG] Fetching key=%s marker=%x last=%x src=%s/%x", key, marker, lid,
		vn.Host, vn.Id[:12])

	return reb.logtrans.FetchKeylog(vn.Host, last, nil)
}

// relocate sends the keys to the new predecessor it needs to takeover.  It returns the
// number of keys relocated and/or an error
func (reb *Relocator) relocate(local, newPred *chord.Vnode) (n int, err error) {
	// Collect keys that need relocating by first calculating the replica id for the key
	// and new pred vnode, then selecting keys who's replica id's are <= to the new
	// predecessor
	out := make([]*KeyLocation, 0)
	reb.idx.Iter(func(key []byte, idx store.KeylogIndex) error {
		// get replica hashes for a key including natural hash
		hashes := hexaring.BuildReplicaHashes(key, int64(reb.replicas), reb.hasher.New())
		// Get location id for key based on local vnode
		rid := getVnodeLocID(local.Id, hashes)
		// Check if replica id is less than our new predecessor and add to list.
		if bytes.Compare(rid, newPred.Id) <= 0 {
			// Try to get last entry otherwise use the marker
			marker := idx.Last()
			if marker == nil {
				marker = idx.Marker()
			}

			kloc := &KeyLocation{Key: key, Marker: marker, Height: idx.Height()}
			out = append(out, kloc)
		}
		return nil
	})

	// No keys to relocate
	if len(out) == 0 {
		return
	}

	// Get relocate stream to remote
	stream, err := reb.trans.GetRelocateStream(local, newPred)
	if err != nil {
		return 0, err
	}
	// Set stream to be re-used.
	defer stream.Recycle()
	// Send selected keys
	for _, kl := range out {
		n++
		if err = stream.Send(kl); err != nil {
			stream.CloseSend()
			return
		}
	}

	err = stream.CloseSend()
	return
}

// start listens to the fetch channel handling each request. It fetches the log for a key
// from the remot in the requets.  This is a blocking call
func (reb *Relocator) start() {

	for req := range reb.fetCh {
		// self is the remote node that has the data.  Key entries will be fetched from this
		// vnode
		src := req.mems.Self
		kl := req.keyloc

		// Fetch the keylog from the remote node
		if _, err := reb.fetch(src, kl.Key, kl.Marker); err != nil {
			log.Printf("[ERROR] Failed to fetch key=%s src=%s/%x error='%v'", kl.Key, src.Host,
				src.Id[:12], err)

		} else {
			log.Printf("[DEBUG] Fetch complete key=%s marker=%x", kl.Key, kl.Marker)
		}

	}
	// signal we have exited the loop
	reb.stopped <- struct{}{}
}

// blocking call
func (reb *Relocator) stop() {
	close(reb.fetCh)
	<-reb.stopped
}
