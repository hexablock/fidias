package fidias

import (
	"errors"
	"fmt"
	"sync"

	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

const (
	// OpSet is the op to set a ke-value pair
	OpSet byte = iota + 1
	// OpDel is the op to delete a key-value pair
	OpDel
)

var (
	errKeyNotFound = errors.New("key not found")
)

// InMemKeyValueFSM is a hexalog FSM for an in-memory key-value store.  It implements the FSM
// interface and provides a get function to retrieve keys as all write are handled by the
// FSM
type InMemKeyValueFSM struct {
	mu sync.RWMutex
	m  map[string]*hexatype.KeyValuePair
}

// NewInMemKeyValueFSM inits a new InMemKeyValueFSM
func NewInMemKeyValueFSM() *InMemKeyValueFSM {
	return &InMemKeyValueFSM{m: make(map[string]*hexatype.KeyValuePair)}
}

// Get gets a value for the key.  It reads it directly from the stored log entry
func (fsm *InMemKeyValueFSM) Get(key []byte) (*hexatype.KeyValuePair, error) {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	value, ok := fsm.m[string(key)]
	if ok {
		return value, nil
	}
	return nil, errKeyNotFound
}

// Apply applies the given entry to the InMemKeyValueFSM.  entryID is the hash id of the entry.
// The first byte in entry.Data contains the operation to be performed followed by the
// actual value.
func (fsm *InMemKeyValueFSM) Apply(entryID []byte, entry *hexatype.Entry) interface{} {
	if entry.Data == nil || len(entry.Data) == 0 {
		return nil
	}

	var (
		op   = entry.Data[0]
		resp interface{}
	)

	switch op {
	case OpSet:
		resp = fsm.applySet(entry.Key, entry.Data[1:], entry)

	case OpDel:
		resp = fsm.applyDelete(string(entry.Key))

	default:
		resp = fmt.Errorf("invalid operation: %x", op)

	}

	return resp
}

func (fsm *InMemKeyValueFSM) applySet(key, value []byte, entry *hexatype.Entry) error {
	kv := &hexatype.KeyValuePair{Entry: entry, Value: value, Key: key}
	fsm.mu.Lock()
	fsm.m[string(key)] = kv
	fsm.mu.Unlock()

	return nil
}

func (fsm *InMemKeyValueFSM) applyDelete(key string) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	if _, ok := fsm.m[key]; !ok {
		return fmt.Errorf("key not found: %s", key)
	}

	delete(fsm.m, key)
	return nil
}

// DummyFSM is a placeholder FSM that does nothing
type DummyFSM struct{}

// Apply gets called by hexalog each time a new entry has been commit and accepted by the
// cluster
func (fsm *DummyFSM) Apply(entryID []byte, entry *hexatype.Entry) interface{} {
	log.Printf("[INFO] DummyFSM key=%s height=%d id=%x data='%s'", entry.Key, entry.Height,
		entryID, entry.Data)
	return nil
}

// Get is a noop to satisfy the KeyValueFSM interface
func (fsm *DummyFSM) Get(key []byte) (*hexatype.KeyValuePair, error) {
	return nil, fmt.Errorf("dummy fsm")
}
