package fidias

import (
	"time"

	chord "github.com/hexablock/go-chord"
)

// NewPredecessor is called when a local vnode finds a new predecessor.  This causes a
// rebalance of keys.  All key hashes less than the new predecessor are transferred to the
// new predecessor.
func (fidias *Fidias) NewPredecessor(local, newPred, oldPred *chord.Vnode) {
	fidias.setLastRingEvent()

	if local.Host == newPred.Host {
		return
	}

	rr := &RebalanceRequest{Src: local, Old: oldPred, Dst: newPred}
	fidias.rebalanceCh <- rr

}

// Leaving is called by the Ring when this node willingly leaves.  This is only
// triggered if an explicit leave is issued
func (fidias *Fidias) Leaving(local, pred, succ *chord.Vnode) {
}

// PredecessorLeaving is only triggered if an explicit leave is issued
func (fidias *Fidias) PredecessorLeaving(local, remote *chord.Vnode) {
	fidias.setLastRingEvent()
}

// SuccessorLeaving is only triggered if an explicit leave is issued
func (fidias *Fidias) SuccessorLeaving(local, remote *chord.Vnode) {
	fidias.setLastRingEvent()
}

// Shutdown is called but a chord node is shutdown
func (fidias *Fidias) Shutdown() {}

func (fidias *Fidias) setLastRingEvent() {
	fidias.tmu.Lock()
	fidias.lastRingEvent = time.Now()
	fidias.tmu.Unlock()
}
