package fidias

import (
	"bytes"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/log"
)

// RebalanceRequest is used to issue a rebalancing of keys.  All key hashes less than that
// of the destination id are transferred to the destination
type RebalanceRequest struct {
	Src *chord.Vnode
	Dst *chord.Vnode
}

func (fidias *Fidias) rebalance(src, dst *chord.Vnode) {

	keys := fidias.keysToTransfer(dst.Id)
	if len(keys) < 1 {
		return
	}

	for key, locID := range keys {

		log.Printf("[DEBUG] Transfer key=%s location-id=%x src=%x dst=%x", key, locID, src.Id, dst.Id)
		if err := fidias.logtrans.TransferKeylog(dst.Host, []byte(key)); err != nil {
			log.Println("[ERROR]", err)
		}

	}

}

// Returns all keys whose location id is less than that of the destination id
func (fidias *Fidias) keysToTransfer(dstID []byte) map[string][]byte {
	// Key to location map
	keys := map[string][]byte{}

	// Iterate through and get all keys and locations that are less than the destination
	fidias.logstore.Iter(func(key string, locationID []byte) {
		// Grab all locations less than the destination
		if bytes.Compare(locationID, dstID) <= 0 {
			keys[key] = locationID
		}
	})

	return keys
}

func (fidias *Fidias) startRebalancing() {

	for {

		select {
		case req := <-fidias.rebalanceCh:
			log.Printf("[INFO] Rebalance %s/%s -> %s/%s",
				req.Src.Host, req.Src.StringID(), req.Dst.Host, req.Dst.StringID())
			fidias.rebalance(req.Src, req.Dst)

		case <-fidias.shutdown:
			return

		}

	}

}
