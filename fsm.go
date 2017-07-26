package fidias

import (
	"fmt"
	"sync"

	"github.com/hexablock/hexalog"
	"github.com/hexablock/log"
)

const (
	opSet byte = iota + 1
	opDel
)

// DummyFSM is a placeholder FSM that does nothing
type DummyFSM struct{}

// Apply gets called by hexalog each time a new entry has been commit and accepted by the
// cluster
func (fsm *DummyFSM) Apply(entry *hexalog.Entry) interface{} {
	log.Printf("[INFO] DummyFSM key=%s height=%d data='%s'", entry.Key, entry.Height, entry.Data)

	return nil
}

// KeyValueItem holds the value and the log entry associated to it
type KeyValueItem struct {
	Key   string
	Value []byte
	Entry *hexalog.Entry
}

// KeyValueFSM is a hexalog FSM for an in-memory key-value store.
type KeyValueFSM struct {
	mu sync.RWMutex
	m  map[string]*KeyValueItem
}

// NewKeyValueFSM inits a new KeyValueFSM
func NewKeyValueFSM() *KeyValueFSM {
	return &KeyValueFSM{m: make(map[string]*KeyValueItem)}
}

// Get gets a value for the key.  It reads it directly from the stored log entry
func (fsm *KeyValueFSM) Get(key string) *KeyValueItem {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	value, _ := fsm.m[key]
	return value
}

// Apply applies the given entry to the KeyValueFSM.  This first byte in the entry data
// contains the operation to be performed.
func (fsm *KeyValueFSM) Apply(entry *hexalog.Entry) interface{} {
	log.Printf("[DEBUG] KeyValueFSM.Apply key=%s height=%d data='%s'", entry.Key, entry.Height, entry.Data)

	if entry.Data == nil || len(entry.Data) == 0 {
		return nil
	}

	var (
		op   = entry.Data[0]
		resp interface{}
	)

	switch op {
	case opSet:
		resp = fsm.applySet(string(entry.Key), entry.Data[1:], entry)

	case opDel:
		resp = fsm.applyDelete(string(entry.Key))

	default:
		resp = fmt.Errorf("invalid operation: %x", op)

	}

	return resp
}

func (fsm *KeyValueFSM) applySet(key string, value []byte, entry *hexalog.Entry) error {
	kv := &KeyValueItem{Entry: entry, Value: value, Key: key}
	fsm.mu.Lock()
	fsm.m[key] = kv
	fsm.mu.Unlock()

	return nil
}

func (fsm *KeyValueFSM) applyDelete(key string) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	if _, ok := fsm.m[key]; !ok {
		return fmt.Errorf("key not found: %s", key)
	}

	delete(fsm.m, key)
	return nil
}
