package fidias

import (
	chord "github.com/hexablock/go-chord"
	"github.com/hexablock/hexatype"
)

// Status contains status information of a node.
type Status struct {
	Version string
	Hash    hexatype.HashAlgorithm
	DHT     *chord.Status
}

// Status returns the status of this node
func (fidias *Fidias) Status() *Status {

	return &Status{
		Version: fidias.conf.Version,
		Hash:    fidias.conf.Hasher().Algorithm(),
		DHT:     fidias.ring.Status(),
	}
}
