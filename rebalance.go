package fidias

import (
	"bytes"
	"sync"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/log"
)

// RebalanceRequest is used to issue a rebalancing of keys.  All key hashes less than that
// of the destination id are transferred to the destination
type RebalanceRequest struct {
	Src *chord.Vnode // Source vnode.  Is a local vnode
	Old *chord.Vnode // The old vnode prior the Dst predecessor
	Dst *chord.Vnode // New vnode to transfer data to
}

// rebalance gets the keys within the range of src and dst and issues transfers requests
// if and as needed.
func (fidias *Fidias) rebalance(src, dst *chord.Vnode) {
	// Get keys in range
	keys := fidias.keysToTransfer(dst.Id)
	if len(keys) < 1 {
		return
	}

	// Transfer keys
	log.Printf("[INFO] Transferring keys=%d src=%x dst=%x", len(keys), src.Id, dst.Id)
	for k, li := range keys {
		// 1 go-routine per key
		go func(key, host string, locID []byte) {

			if err := fidias.trans.remote.TransferKeylog(host, []byte(key)); err != nil {
				log.Printf("[ERROR] Failed to transfer location-id=%x key=%s error='%v'", locID, key, err)
			}

		}(k, dst.Host, li)

	}

}

// Returns all keys whose location id is less than that of the destination id
func (fidias *Fidias) keysToTransfer(dstID []byte) map[string][]byte {
	// Key to location map
	keys := map[string][]byte{}

	// Iterate through and get all keys and locations that are less than the destination
	fidias.trans.local.Iter(func(key string, locationID []byte) {
		// Grab all locations less than the destination
		if bytes.Compare(locationID, dstID) <= 0 {
			keys[key] = locationID
		}
	})

	return keys
}

func (fidias *Fidias) startRebalancer() {

	for req := range fidias.rebalanceCh {
		log.Printf("[INFO] Rebalance %s/%s -> %s/%s", req.Src.Host, req.Src.StringID(), req.Dst.Host, req.Dst.StringID())
		fidias.rebalance(req.Src, req.Dst)
	}

	fidias.shutdown <- struct{}{}
}

type rebalancer struct {
	rmu sync.RWMutex
	r   map[string]struct{} // keys being recieved
	tmu sync.RWMutex
	t   map[string]struct{} // keys being transfered
}

func newRebalancer() *rebalancer {
	return &rebalancer{
		r: make(map[string]struct{}),
		t: make(map[string]struct{}),
	}
}

func (reb *rebalancer) isReceiving(key string) bool {
	reb.rmu.RLock()
	defer reb.rmu.RUnlock()
	_, ok := reb.r[key]
	return ok
}

func (reb *rebalancer) setReceive(key string) {
	reb.rmu.Lock()
	reb.r[key] = struct{}{}
	reb.rmu.Unlock()
}

func (reb *rebalancer) unsetReceive(key string) {
	reb.rmu.Lock()
	if _, ok := reb.r[key]; ok {
		delete(reb.r, key)
	}
	reb.rmu.Unlock()
}

func (reb *rebalancer) isTransfering(key string) bool {
	reb.tmu.RLock()
	defer reb.tmu.RUnlock()
	_, ok := reb.t[key]
	return ok
}

func (reb *rebalancer) setTransfer(keys ...string) {
	reb.tmu.Lock()
	for _, k := range keys {
		reb.t[k] = struct{}{}
	}
	reb.tmu.Unlock()
}

func (reb *rebalancer) unsetTransfer(keys ...string) {
	reb.tmu.Lock()
	for _, k := range keys {
		if _, ok := reb.t[k]; ok {
			delete(reb.t, k)
		}
	}
	reb.tmu.Unlock()
}
