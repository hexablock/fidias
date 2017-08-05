package fidias

import (
	"errors"
	"fmt"
	"sync"

	"github.com/hexablock/hexalog"
	"github.com/hexablock/log"
)

const (
	opSet byte = iota + 1
	opDel
)

var (
	errKeyNotFound = errors.New("key not found")
)

// KeyValuePair holds the value and the log entry associated to it
// type KeyValuePair struct {
// 	Key   string
// 	Value []byte
// 	Entry *hexalog.Entry
// }

// InMemKeyValueFSM is a hexalog FSM for an in-memory key-value store.  It implements the FSM
// interface and provides a get function to retrieve keys as all write are handled by the
// FSM
type InMemKeyValueFSM struct {
	mu sync.RWMutex
	m  map[string]*KeyValuePair
}

// NewInMemKeyValueFSM inits a new InMemKeyValueFSM
func NewInMemKeyValueFSM() *InMemKeyValueFSM {
	return &InMemKeyValueFSM{m: make(map[string]*KeyValuePair)}
}

// Get gets a value for the key.  It reads it directly from the stored log entry
func (fsm *InMemKeyValueFSM) Get(key []byte) (*KeyValuePair, error) {
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
func (fsm *InMemKeyValueFSM) Apply(entryID []byte, entry *hexalog.Entry) interface{} {
	if entry.Data == nil || len(entry.Data) == 0 {
		return nil
	}

	var (
		op   = entry.Data[0]
		resp interface{}
	)

	switch op {
	case opSet:
		resp = fsm.applySet(entry.Key, entry.Data[1:], entry)

	case opDel:
		resp = fsm.applyDelete(string(entry.Key))

	default:
		resp = fmt.Errorf("invalid operation: %x", op)

	}

	return resp
}

func (fsm *InMemKeyValueFSM) applySet(key, value []byte, entry *hexalog.Entry) error {
	kv := &KeyValuePair{Entry: entry, Value: value, Key: key}
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
func (fsm *DummyFSM) Apply(entryID []byte, entry *hexalog.Entry) interface{} {
	log.Printf("[INFO] DummyFSM key=%s height=%d id=%x data='%s'", entry.Key, entry.Height,
		entryID, entry.Data)
	return nil
}

// Get is a noop to satisfy the KeyValueFSM interface
func (fsm *DummyFSM) Get(key []byte) (*KeyValuePair, error) {
	return nil, fmt.Errorf("dummy fsm")
}
