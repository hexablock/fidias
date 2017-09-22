package fidias

import (
	"bytes"
	"sync"
	"time"

	chord "github.com/hexablock/go-chord"
)

// keyblock is a block of keys around the ring between the left and right hashes including
// the right hash.  This is the vnode and predecessor id vica-versa
type keyBlock struct {
	pred  *chord.Vnode
	local *chord.Vnode
}

// isMultiHost returns whether pred and local span across 2 nodes
func (kb *keyBlock) isMultiHost() bool {
	return kb.pred.Host != kb.local.Host
}

// Checks if a key is between pred and local, local inclusive (right inclusive)
func (kb *keyBlock) owns(hash []byte) bool {
	// Check for ring wrap around
	if bytes.Compare(kb.pred.Id, kb.local.Id) == 1 {
		return bytes.Compare(kb.pred.Id, hash) == -1 ||
			bytes.Compare(kb.local.Id, hash) >= 0
	}

	return bytes.Compare(kb.pred.Id, hash) == -1 &&
		bytes.Compare(kb.local.Id, hash) >= 0
}

type keyBlockSet struct {
	mu             sync.RWMutex
	m              map[string]*keyBlock
	lastChangeTime time.Time
}

func newKeyBlockSet() *keyBlockSet {
	return &keyBlockSet{
		m: make(map[string]*keyBlock),
	}
}

func (kbr *keyBlockSet) get(localID []byte) *keyBlock {
	kbr.mu.RLock()
	defer kbr.mu.RUnlock()

	return kbr.m[string(localID)]
}

// set sets a keyblock using the local id as the key.
func (kbr *keyBlockSet) set(pred, local *chord.Vnode) {
	kbr.mu.Lock()
	kbr.m[string(local.Id)] = &keyBlock{pred: pred, local: local}
	kbr.lastChangeTime = time.Now()
	kbr.mu.Unlock()
}

func (kbr *keyBlockSet) unset(pred, local *chord.Vnode) {
	k := string(local.Id)

	kbr.mu.Lock()
	if kb, ok := kbr.m[k]; ok {
		if bytes.Compare(kb.pred.Id, pred.Id) == 0 {
			delete(kbr.m, k)
			kbr.lastChangeTime = time.Now()
		}
	}
	kbr.mu.Unlock()
}

func (kbr *keyBlockSet) lastChange() time.Time {
	kbr.mu.Lock()
	defer kbr.mu.Unlock()

	return kbr.lastChangeTime
}

// owns returns true if any of the local vnodes own the given hash
func (kbr *keyBlockSet) owns(hash []byte) bool {
	kbr.mu.Lock()
	defer kbr.mu.Unlock()

	for _, kb := range kbr.m {
		if kb.owns(hash) {
			return true
		}
	}

	return false
}

func (kbr *keyBlockSet) blockCount() int {
	kbr.mu.RLock()
	defer kbr.mu.RUnlock()
	return len(kbr.m)
}
