package fidias

import "github.com/hexablock/hexatype"

// KeyValueStore implements a key value store interface
type KeyValueStore interface {
	Get(key []byte) (*hexatype.KeyValuePair, error)
}
