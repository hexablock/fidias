package fidias

import (
	"fmt"

	"github.com/hexablock/hexalog"
)

const (
	// OpSet is the op to set a ke-value pair
	OpSet byte = iota + 1
	// OpDel is the op to delete a key-value pair
	OpDel
)

// KeyValueFSM is an FSM for a key value store.  Aside from fsm functions,
// it also contains read-only key-value functions needed.
type KeyValueFSM interface {
	// Get a key
	Get(key []byte) (*KeyValuePair, error)
	// Apply a set operation entry with value containing the data
	ApplySet(entryID []byte, entry *hexalog.Entry, value []byte) error
	// Apply a delete entry
	ApplyDelete(entry *hexalog.Entry) error
}

// FSM is a hexalog FSM for an in-memory key-value store.  It implements the
// FSM interface and provides a get function to retrieve keys as all write
// are handled by the FSM
type FSM struct {
	kv KeyValueFSM // FSM for key-value pairs
}

// NewFSM inits a new FSM
func NewFSM(kvprefix, fsprefix string) *FSM {
	return &FSM{
		kv: NewInMemKeyValueFSM(kvprefix),
	}
}

// Open initialized the internal data structures.  It always returns nil
func (fsm *FSM) Open() error {
	return nil
}

// GetKey gets a value for the key.  It reads it directly from the stored log
// entry
func (fsm *FSM) GetKey(key []byte) (*KeyValuePair, error) {
	return fsm.kv.Get(key)
}

// Apply applies the given entry to the FSM.  entryID is the hash id of the
// entry.  The first byte in entry.Data contains the operation to be performed
// followed by the actual value.
func (fsm *FSM) Apply(entryID []byte, entry *hexalog.Entry) interface{} {
	if entry.Data == nil || len(entry.Data) == 0 {
		return nil
	}

	var (
		op   = entry.Data[0]
		resp interface{}
	)

	switch op {
	case OpSet:
		resp = fsm.kv.ApplySet(entryID, entry, entry.Data[1:])

	case OpDel:
		resp = fsm.kv.ApplyDelete(entry)

	default:
		resp = fmt.Errorf("invalid operation: %x", op)

	}

	return resp
}

// Close is a no-op to satisfy the KeyValueFSM interface
func (fsm *FSM) Close() error {
	return nil
}
