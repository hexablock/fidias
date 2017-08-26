package fidias

import (
	"github.com/hexablock/go-chord"
	"github.com/hexablock/log"
)

// NewPredecessor is called when a local vnode finds a new predecessor.  This causes a
// rebalance of keys.  All key hashes less than the new predecessor are transferred to the
// new predecessor.
func (fidias *Fidias) NewPredecessor(local, newPred, oldPred *chord.Vnode) {
	fidias.keyblocks.set(newPred, local)

	// local-to-local rebalance.  Handle rebalancing data on the same local node
	if local.Host == newPred.Host {
		return
	}

	// Send keys that need to be relocated.  This is a blocking call.
	n, rt, err := fidias.rel.relocate(local, newPred)
	if err != nil {
		log.Printf("[ERROR] Relocation incomplete error='%v' src=%s/%x dst=%s/%x runtime=%v", err,
			local.Host, local.Id[:12], newPred.Host, newPred.Id[:12], rt)

	} else {
		log.Printf("[INFO] Relocation complete keys=%d src=%s/%x dst=%s/%x runtime=%v", n, local.Host,
			local.Id[:12], newPred.Host, newPred.Id[:12], rt)
	}

}

// Leaving is called by the Ring when this node willingly leaves.  This is only
// triggered if an explicit leave is issued
func (fidias *Fidias) Leaving(local, pred, succ *chord.Vnode) {
}

// PredecessorLeaving is only triggered if an explicit leave is issued
func (fidias *Fidias) PredecessorLeaving(local, remote *chord.Vnode) {
	// unset the remote link to the local vnode
	fidias.keyblocks.unset(remote, local)
}

// SuccessorLeaving is only triggered if an explicit leave is issued
func (fidias *Fidias) SuccessorLeaving(local, remote *chord.Vnode) {
	//fidias.setLastRingEvent()
}

// Shutdown is called but a chord node is shutdown
func (fidias *Fidias) Shutdown() {}
