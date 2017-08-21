package fidias

import chord "github.com/hexablock/go-chord"

// Status contains status information of a node.
type Status struct {
	Ring *chord.Status
}

// Status returns the status of this node
func (fidias *Fidias) Status() *Status {
	return &Status{
		Ring: fidias.ring.Status(),
	}
}
