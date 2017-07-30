package fidias

import (
	"bytes"

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
	for key, locID := range keys {
		log.Printf("[DEBUG] Transfer location-id=%x key=%s src=%x dst=%x", locID, key, src.Id, dst.Id)
		if err := fidias.logtrans.TransferKeylog(dst.Host, []byte(key)); err != nil {
			log.Printf("[ERROR] Failed to transfer location-id=%x key=%s error='%v'", locID, key, err)
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

func (fidias *Fidias) start() {
	// Get the heal channel from the log
	healCh := fidias.hexlog.Heal()

	for {

		select {
		case req := <-fidias.rebalanceCh:
			log.Printf("[INFO] Rebalance %s/%s -> %s/%s",
				req.Src.Host, req.Src.StringID(), req.Dst.Host, req.Dst.StringID())

			fidias.rebalance(req.Src, req.Dst)

		case req := <-healCh:
			if _, _, err := fidias.heal(req); err != nil {
				log.Printf("[ERROR] Failed to heal key=%s height=%d id=%x error='%v'",
					req.Entry.Key, req.Entry.Height, req.ID, err)
			}

		case <-fidias.shutdown:
			return

		}

	}
	// end
}
