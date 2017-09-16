package fidias

import (
	"fmt"
	"sync"

	"github.com/dgraph-io/badger"
	"github.com/golang/protobuf/proto"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

const (
	// OpSet is the op to set a ke-value pair
	OpSet byte = iota + 1
	// OpDel is the op to delete a key-value pair
	OpDel
)

// InMemKeyValueFSM is a hexalog FSM for an in-memory key-value store.  It implements the
// FSM interface and provides a get function to retrieve keys as all write are handled by
// the FSM
type InMemKeyValueFSM struct {
	mu sync.RWMutex
	m  map[string]*hexatype.KeyValuePair
}

// NewInMemKeyValueFSM inits a new InMemKeyValueFSM
func NewInMemKeyValueFSM() *InMemKeyValueFSM {
	return &InMemKeyValueFSM{}
}

// Open initialized the internal data structures.  It always returns nil
func (fsm *InMemKeyValueFSM) Open() error {
	fsm.m = make(map[string]*hexatype.KeyValuePair)
	return nil
}

// GetKey gets a value for the key.  It reads it directly from the stored log entry
func (fsm *InMemKeyValueFSM) GetKey(key []byte) (*hexatype.KeyValuePair, error) {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	value, ok := fsm.m[string(key)]
	if ok {
		return value, nil
	}
	return nil, hexatype.ErrKeyNotFound
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

// Close is a no-op to satisfy the KeyValueFSM interface
func (fsm *InMemKeyValueFSM) Close() error {
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

// Open is a no-op to satisfy the KeyValueFSM interface
func (fsm *DummyFSM) Open() error {
	return nil
}

// GetKey is a noop to satisfy the KeyValueFSM interface
func (fsm *DummyFSM) GetKey(key []byte) (*hexatype.KeyValuePair, error) {
	return nil, fmt.Errorf("dummy fsm")
}

// Close is a no-op to satisfy the KeyValueFSM interface
func (fsm *DummyFSM) Close() error {
	return nil
}

// BadgerKeyValueFSM implements a badger backed KeyValueFSM
type BadgerKeyValueFSM struct {
	opt *badger.Options
	kv  *badger.KV
}

// NewBadgerKeyValueFSM inits a new BadgerKeyValueFSM
func NewBadgerKeyValueFSM(dataDir string) *BadgerKeyValueFSM {
	opt := new(badger.Options)
	*opt = badger.DefaultOptions
	opt.Dir = dataDir
	opt.ValueDir = dataDir

	// TODO: handle this a better way
	opt.SyncWrites = true

	return &BadgerKeyValueFSM{opt: opt}
}

// Open opens the store for reading and writing.  This must be called before the FSM
// can be used.
func (fsm *BadgerKeyValueFSM) Open() (err error) {
	fsm.kv, err = badger.NewKV(fsm.opt)
	return
}

// Get gets a value for the key.  It reads it directly from the stored log entry
func (fsm *BadgerKeyValueFSM) GetKey(key []byte) (*hexatype.KeyValuePair, error) {
	var item badger.KVItem
	err := fsm.kv.Get(key, &item)
	if err != nil {
		return nil, err
	}

	var val []byte

	err = item.Value(func(v []byte) error {
		if v != nil {
			val = make([]byte, len(v))
			copy(val, v)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, hexatype.ErrKeyNotFound
	}

	var kvp hexatype.KeyValuePair
	err = proto.Unmarshal(val, &kvp)
	return &kvp, err
}

// Apply applies the given entry to the BadgerKeyValueFSM.  entryID is the hash id of the entry.
// The first byte in entry.Data contains the operation to be performed followed by the
// actual value.
func (fsm *BadgerKeyValueFSM) Apply(entryID []byte, entry *hexatype.Entry) interface{} {
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
		resp = fsm.applyDelete(entry.Key)

	default:
		resp = fmt.Errorf("invalid operation: %x", op)

	}

	return resp
}

func (fsm *BadgerKeyValueFSM) applySet(key, value []byte, entry *hexatype.Entry) error {
	kv := &hexatype.KeyValuePair{Entry: entry, Value: value, Key: key}
	val, err := proto.Marshal(kv)
	if err == nil {
		err = fsm.kv.Set(key, val, 0)
	}

	return err
}

func (fsm *BadgerKeyValueFSM) applyDelete(key []byte) error {
	return fsm.kv.Delete(key)
}

// Close closes the underlying badger store
func (fsm *BadgerKeyValueFSM) Close() error {
	return fsm.kv.Close()
}
