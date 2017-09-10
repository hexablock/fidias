package fidias

import (
	"bytes"
	"time"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog/store"
	"github.com/hexablock/hexaring"
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
	conf  *Config
	idx   store.IndexStore
	trans RelocatorTransport
}

// NewRelocator instantiates a new Relocator
func NewRelocator(conf *Config, idx store.IndexStore, trans RelocatorTransport) *Relocator {

	return &Relocator{
		conf:  conf,
		idx:   idx,
		trans: trans,
	}

}

// relocate sends the keys to the new predecessor it needs to takeover.  It returns the
// number of keys relocated and/or an error
func (reb *Relocator) relocate(local, newPred *chord.Vnode) (n int, rt time.Duration, err error) {
	// Collect keys that need relocating by first calculating the replica id for the key
	// and new pred vnode, then selecting keys who's replica id's are <= to the new
	// predecessor
	start := time.Now()
	out := make([]*KeyLocation, 0)
	reb.idx.Iter(func(key []byte, idx store.KeylogIndex) error {
		// get replica hashes for a key including natural hash
		hashes := hexaring.BuildReplicaHashes(key, int64(reb.conf.Hexalog.Votes), reb.conf.Hasher().New())
		// Get location id for key based on local vnode
		rid := getVnodeLocID(local.Id, hashes)

		// Check if replica id is less than our new predecessor and add to list.
		if bytes.Compare(rid, newPred.Id) <= 0 {
			// Try to get last entry otherwise use the marker
			marker := idx.Last()
			if marker == nil {
				marker = idx.Marker()
			}

			kloc := &KeyLocation{
				Key:    key,
				Marker: marker,
				Height: idx.Height(),
			}

			out = append(out, kloc)
		}

		// Move onto to the next
		return nil
	})

	// No keys to relocate
	if len(out) == 0 {
		rt = time.Since(start)
		return
	}

	// Get relocate stream to remote
	stream, err := reb.trans.GetRelocateStream(local, newPred)
	if err != nil {
		rt = time.Since(start)
		return 0, rt, err
	}
	// Set stream to be re-used.
	defer stream.Recycle()
	// Send selected keys
	for _, kl := range out {
		n++
		if err = stream.Send(kl); err != nil {
			stream.CloseSend()
			rt = time.Since(start)
			return
		}
	}

	err = stream.CloseSend()
	rt = time.Since(start)
	return
}
