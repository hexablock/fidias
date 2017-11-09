package fidias

import (
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/memberlist"

	"github.com/hexablock/go-kelips"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
	"github.com/hexablock/vivaldi"
)

const (
	msgTypeInsertKey = iota + 3
	msgTypePurgeKey
)

type delegate struct {
	// Local node
	local hexatype.Node

	// Coordinate for virtual positioning
	coord *vivaldi.Client

	// DHT
	dht *kelips.Kelips

	// Message broadcast buffer
	mu         sync.RWMutex
	broadcasts [][]byte
}

func (del *delegate) NotifyConflict(n1 *memberlist.Node, n2 *memberlist.Node) {
	log.Println("NotifyConflict", n1, n2)
}

func (del *delegate) NotifyAlive(node *memberlist.Node) error {
	return nil
}

// NodeMeta is used to retrieve meta-data about the current node
// when broadcasting an alive message. It's length is limited to
// the given byte size. This metadata is available in the Node structure.
func (del *delegate) NodeMeta(limit int) []byte {
	node := del.local
	node.Coordinates = del.coord.GetCoordinate()

	b, err := proto.Marshal(&node)
	if err == nil {
		log.Printf("[DEBUG] NodeMeta: size=%d/%d", len(b), limit)
		return b
	}
	log.Printf("[ERROR] NodeMeta: %v", err)
	return nil
}

// NotifyMsg is called when a user-data message is received.
// Care should be taken that this method does not block, since doing
// so would block the entire UDP packet receive loop. Additionally, the byte
// slice may be modified after the call returns, so it should be copied if
// needed
func (del *delegate) NotifyMsg(msg []byte) {

	var (
		err error
		typ = msg[0]
	)

	switch typ {
	case msgTypeInsertKey:
		tuple := kelips.TupleHost(msg[1:19])
		key := msg[19:]
		// Perform a single insert
		err = del.dht.Insert(key, tuple)

	default:
		log.Println("[DEBUG] NotifyMsg:", msg)
	}

	if err != nil {
		log.Println("[ERROR] NotifyMsg", err)
	}

}

// LocalState is used for a TCP Push/Pull. This is sent to
// the remote side in addition to the membership information. Any
// data can be sent here. See MergeRemoteState as well. The `join`
// boolean indicates this is for a join instead of a push/pull.
func (del *delegate) LocalState(join bool) []byte {
	if join {

		snapshot := del.dht.Snapshot()
		b, err := proto.Marshal(snapshot)
		if err != nil {
			log.Println("[ERROR]", err)
			return nil
		}

		return b
	}

	//	log.Println("[TODO] Perform a state diff with remote", node.Host())

	return nil
}

// MergeRemoteState is invoked after a TCP Push/Pull. This is the
// state received from the remote side and is the result of the
// remote side's LocalState call. The 'join'
// boolean indicates this is for a join instead of a push/pull.
func (del *delegate) MergeRemoteState(buf []byte, join bool) {

	if join {
		del.seedDHT(buf)
	}

}

// GetBroadcasts is called when user data messages can be broadcast.
// It can return a list of buffers to send. Each buffer should assume an
// overhead as provided with a limit on the total byte size allowed.
// The total byte size of the resulting data to send must not exceed
// the limit. Care should be taken that this method does not block,
// since doing so would block the entire UDP packet receive loop.
func (del *delegate) GetBroadcasts(overhead, limit int) [][]byte {
	del.mu.RLock()
	if len(del.broadcasts) < 1 {
		del.mu.RUnlock()
		return nil
	}

	out := make([][]byte, len(del.broadcasts))
	copy(out, del.broadcasts)
	del.mu.RUnlock()

	// TODO: Check size
	//maxsize := limit - overhead

	del.mu.Lock()
	del.broadcasts = make([][]byte, 0)
	del.mu.Unlock()

	return out
}

func (del *delegate) seedDHT(buf []byte) {
	// Unmarshal snapshot
	var ss kelips.Snapshot
	err := proto.Unmarshal(buf, &ss)
	if err != nil {
		log.Println("[ERROR]", err)
		return
	}

	if err = del.dht.Seed(&ss); err != nil {
		log.Println("[ERROR] Failed to seed snapshot:", err)
		return
	}

	log.Printf("[INFO] DHT seeded tuples=%d nodes=%d", len(ss.Tuples), len(ss.Nodes))
}
