package fidias

import (
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexatype"
)

// SimpleJury implements a Jury interface using kelips dht as the backend.  It
// returns the first n nodes available in a group
type SimpleJury struct {
	dht DHT
}

// NewSimpleJury inits a new Kelips DHT backed jury
func NewSimpleJury(dht DHT) *SimpleJury {
	return &SimpleJury{dht: dht}
}

// Participants gets the AffinityGroup group for the key and returns the nodes
// in that group as participants
func (jury *SimpleJury) Participants(key []byte, min int) ([]*hexalog.Participant, error) {
	nodes, err := jury.dht.LookupGroupNodes(key)
	if err != nil {
		return nil, err
	}
	if len(nodes) < min {
		return nil, hexatype.ErrInsufficientPeers
	}

	participants := make([]*hexalog.Participant, 0, len(nodes))
	for i, n := range nodes {
		pcp := jury.participantFromNode(n, int32(i), 0)
		participants = append(participants, pcp)
	}

	return participants, nil
}

func (jury *SimpleJury) participantFromNode(n *hexatype.Node, p int32, i int32) *hexalog.Participant {
	meta := n.Metadata()

	return &hexalog.Participant{
		ID:       n.ID,
		Host:     meta["hexalog"],
		Priority: p,
		Index:    i,
	}
}
