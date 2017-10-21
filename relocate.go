package fidias

import (
	"bytes"
	"time"

	"github.com/hexablock/blox/device"
	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

// RelocatorTransport implements a transport needed by the key rebalancing engine
type RelocatorTransport interface {
	GetRelocateStream(local, remote *chord.Vnode) (*RelocateStream, error)
	GetRelocateBlocksStream(local, remote *chord.Vnode) (*RelocateBlocksStream, error)
}

// rebReq contains data for perform a relocation
type relocateReq struct {
	keyloc *KeyState
	mems   *chord.VnodePair
}

// Relocator is responsible for moving data as needed when the underlying cluster topology
// changes
type Relocator struct {
	// This is needed to compute relocation
	replicas int64
	hasher   hexatype.Hasher
	// Keylog index
	idx hexalog.IndexStore
	// Block index
	blkj device.Journal
	// RPC transport
	trans RelocatorTransport
}

// NewRelocator instantiates a new Relocator
func NewRelocator(replicas int64, hasher hexatype.Hasher) *Relocator {

	return &Relocator{
		replicas: replicas,
		hasher:   hasher,
	}

}

// RegisterTransport registers the transport to be used for relocation
func (reb *Relocator) RegisterTransport(trans RelocatorTransport) {
	reb.trans = trans
}

// RegisterBlockJournal registers a block journal to the relocator to be used to determine
// which blocks need to be relocated.
func (reb *Relocator) RegisterBlockJournal(journal device.Journal) {
	reb.blkj = journal
}

// RegisterKeylogIndex register an index store of keylogs  to the relocator to be used to
// determine the keys that need to be relocated
func (reb *Relocator) RegisterKeylogIndex(idx hexalog.IndexStore) {
	reb.idx = idx
}

func (reb *Relocator) relocate(local, newPred *chord.Vnode) (n int, rt time.Duration, err error) {
	// Relocate hexalog keys
	n, rt, err = reb.relocateKeylogs(local, newPred)

	// Do block relocation in a go-routine after keylocg relocation as it's not as critical
	// to get the block id's across
	go func() {

		c, brt, er := reb.relocateBlocks(local, newPred)
		if er != nil {
			log.Printf("[ERROR] Relocate blocks failed src=%s/%x dst=%s/%x error='%v'",
				local.Host, local.Id[:12], newPred.Host, newPred.Id[:12], er)
		} else if c > 0 {
			log.Printf("[INFO] Relocated blocks=%d src=%s/%x dst=%s/%x runtime=%v",
				c, local.Host, local.Id[:12], newPred.Host, newPred.Id[:12], brt)
		}

	}()
	// Return data for hexalog relocations
	return n, rt, err
}

// relocateKeylogs sends the keys to the new predecessor it needs to takeover.  It returns the
// number of keys relocated and/or an error
func (reb *Relocator) relocateKeylogs(local, newPred *chord.Vnode) (n int, rt time.Duration, err error) {
	// Collect keys that need relocating by first calculating the replica id for the key
	// and new pred vnode, then selecting keys who's replica id's are <= to the new
	// predecessor
	start := time.Now()
	out := make([]*KeyState, 0)
	// This obtains a read lock.
	reb.idx.Iter(func(key []byte, idx hexalog.KeylogIndex) error {
		// get replica hashes for a key including natural hash
		hashes := hexaring.BuildReplicaHashes(key, reb.replicas, reb.hasher.New())
		// Get location id for key based on local vnode
		rid := getVnodeLocID(local.Id, hashes)

		// Check if replica id is less than our new predecessor and add to list.
		if bytes.Compare(rid, newPred.Id) <= 0 {
			// Try to get last entry otherwise use the marker
			marker := idx.Last()
			if marker == nil {
				marker = idx.Marker()
			}

			kloc := &KeyState{
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

// relocateBlocks sends the block ids to the new predecessor it needs to takeover.  It
// returns the number of blocks relocated and/or an error
func (reb *Relocator) relocateBlocks(local, newPred *chord.Vnode) (n int, rt time.Duration, err error) {
	// Collect keys that need relocating by first calculating the replica id for the key
	// and new pred vnode, then selecting keys who's replica id's are <= to the new
	// predecessor
	start := time.Now()
	out := make([]*KeyState, 0)
	// This obtains a read lock.
	reb.blkj.Iter(func(jent *device.JournalEntry) error {
		// Get replica hashes for a key including natural hash
		hashes := hexaring.BuildReplicaHashes(jent.ID(), reb.replicas, reb.hasher.New())
		// Get location id for key based on local vnode
		rid := getVnodeLocID(local.Id, hashes)
		// Check if replica id is less than our new predecessor and add to list.
		if bytes.Compare(rid, newPred.Id) <= 0 {

			kloc := &KeyState{
				Key:    jent.ID(),
				Marker: append([]byte{byte(jent.Type())}, jent.Data()...),
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
	stream, err := reb.trans.GetRelocateBlocksStream(local, newPred)
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
