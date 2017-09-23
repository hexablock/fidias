package fidias

import (
	chord "github.com/hexablock/go-chord"
	"github.com/hexablock/hexaring"
)

type DHT interface {
	LookupReplicated(key []byte, replicas int) (hexaring.LocationSet, error)
	LookupReplicatedHash(hash []byte, replicas int) (hexaring.LocationSet, error)
	ScourReplicatedKey(key []byte, replicas int, cb func(*chord.Vnode) error) (int, error)
}