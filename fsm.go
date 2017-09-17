package fidias

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/hexablock/hexatype"
)

const (
	// OpSet is the op to set a ke-value pair
	OpSet byte = iota + 1
	// OpDel is the op to delete a key-value pair
	OpDel
)

const (
	// OpFsSet is used to set a path in the fsm
	OpFsSet byte = iota + 10
	// OpFsDel is used to delete a path from the fsm
	OpFsDel
)

// InMemFSM is a hexalog FSM for an in-memory key-value store.  It implements the
// FSM interface and provides a get function to retrieve keys as all write are handled by
// the FSM
type InMemFSM struct {
	kvprefix []byte
	fsprefix []byte

	// key-value pairs
	kvLock sync.RWMutex
	kv     map[string]*KeyValuePair
	// fs
	fsLock sync.RWMutex
	fs     map[string]*Versioned
}

// NewInMemFSM inits a new InMemFSM
func NewInMemFSM(kvprefix, fsprefix string) *InMemFSM {
	return &InMemFSM{kvprefix: []byte(kvprefix), fsprefix: []byte(fsprefix)}
}

// Open initialized the internal data structures.  It always returns nil
func (fsm *InMemFSM) Open() error {
	fsm.kv = make(map[string]*KeyValuePair)
	fsm.fs = make(map[string]*Versioned)
	return nil
}

// GetKey gets a value for the key.  It reads it directly from the stored log entry
func (fsm *InMemFSM) GetKey(key []byte) (*KeyValuePair, error) {
	fsm.kvLock.RLock()
	defer fsm.kvLock.RUnlock()

	value, ok := fsm.kv[string(key)]
	if ok {
		return value, nil
	}
	return nil, hexatype.ErrKeyNotFound
}

// GetPath returns a path with pointers to all of its versions
func (fsm *InMemFSM) GetPath(name string) (*Versioned, error) {
	fsm.fsLock.RLock()
	defer fsm.fsLock.RUnlock()

	value, ok := fsm.fs[name]
	if ok {
		return value, nil
	}
	return nil, fmt.Errorf("path not found: %s", name)
}

// Apply applies the given entry to the InMemFSM.  entryID is the hash id of the entry.
// The first byte in entry.Data contains the operation to be performed followed by the
// actual value.
func (fsm *InMemFSM) Apply(entryID []byte, entry *hexatype.Entry) interface{} {
	if entry.Data == nil || len(entry.Data) == 0 {
		return nil
	}

	var (
		op   = entry.Data[0]
		resp interface{}
	)

	switch op {
	case OpSet:
		resp = fsm.applySet(entry)

	case OpDel:
		resp = fsm.applyDelete(entry)

	case OpFsSet:
		resp = fsm.applyFSSet(entry)

	case OpFsDel:
		resp = fsm.applyFSDelete(entry)

	default:
		resp = fmt.Errorf("invalid operation: %x", op)

	}

	return resp
}

func (fsm *InMemFSM) applyFSSet(entry *hexatype.Entry) error {
	key := bytes.TrimPrefix(entry.Key, fsm.fsprefix)
	ver := NewVersioned(key)

	value := entry.Data[1:]
	if err := ver.UnmarshalBinary(value); err != nil {
		return err
	}

	fsm.fsLock.Lock()
	fsm.fs[string(key)] = ver
	fsm.fsLock.Unlock()

	return nil
}

func (fsm *InMemFSM) applyFSDelete(entry *hexatype.Entry) error {
	key := string(bytes.TrimPrefix(entry.Key, fsm.fsprefix))

	fsm.fsLock.Lock()
	defer fsm.fsLock.Unlock()

	if _, ok := fsm.fs[key]; !ok {
		return fmt.Errorf("path not found: %s", key)
	}

	delete(fsm.fs, key)
	return nil
}

func (fsm *InMemFSM) applySet(entry *hexatype.Entry) error {
	key := bytes.TrimPrefix(entry.Key, fsm.kvprefix)
	value := entry.Data[1:]

	kv := &KeyValuePair{Entry: entry, Value: value, Key: key}

	fsm.kvLock.Lock()
	fsm.kv[string(key)] = kv
	fsm.kvLock.Unlock()

	return nil
}

func (fsm *InMemFSM) applyDelete(entry *hexatype.Entry) error {
	key := string(bytes.TrimPrefix(entry.Key, fsm.kvprefix))

	fsm.kvLock.Lock()
	defer fsm.kvLock.Unlock()

	if _, ok := fsm.kv[key]; !ok {
		return fmt.Errorf("key not found: %s", key)
	}

	delete(fsm.kv, key)
	return nil
}

// Close is a no-op to satisfy the KeyValueFSM interface
func (fsm *InMemFSM) Close() error {
	return nil
}

// BadgerKeyValueFSM implements a badger backed KeyValueFSM
// type BadgerKeyValueFSM struct {
// 	opt *badger.Options
// 	kv  *badger.KV
// }

// NewBadgerKeyValueFSM inits a new BadgerKeyValueFSM
// func NewBadgerKeyValueFSM(dataDir string) *BadgerKeyValueFSM {
// 	opt := new(badger.Options)
// 	*opt = badger.DefaultOptions
// 	opt.Dir = dataDir
// 	opt.ValueDir = dataDir

// 	// TODO: handle this a better way
// 	opt.SyncWrites = true

// 	return &BadgerKeyValueFSM{opt: opt}
// }

// Open opens the store for reading and writing.  This must be called before the FSM
// can be used.
// func (fsm *BadgerKeyValueFSM) Open() (err error) {
// 	fsm.kv, err = badger.NewKV(fsm.opt)
// 	return
// }

// GetKey gets a value for the key.  It reads it directly from the stored log entry
// func (fsm *BadgerKeyValueFSM) GetKey(key []byte) (*KeyValuePair, error) {
// 	var item badger.KVItem
// 	err := fsm.kv.Get(key, &item)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var val []byte

// 	err = item.Value(func(v []byte) error {
// 		if v != nil {
// 			val = make([]byte, len(v))
// 			copy(val, v)
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	if val == nil {
// 		return nil, hexatype.ErrKeyNotFound
// 	}

// 	var kvp KeyValuePair
// 	err = proto.Unmarshal(val, &kvp)
// 	return &kvp, err
// }

// Apply applies the given entry to the BadgerKeyValueFSM.  entryID is the hash id of the entry.
// The first byte in entry.Data contains the operation to be performed followed by the
// actual value.
// func (fsm *BadgerKeyValueFSM) Apply(entryID []byte, entry *hexatype.Entry) interface{} {
// 	if entry.Data == nil || len(entry.Data) == 0 {
// 		return nil
// 	}

// 	var (
// 		op   = entry.Data[0]
// 		resp interface{}
// 	)

// 	switch op {
// 	case OpSet:
// 		resp = fsm.applySet(entry)

// 	case OpDel:
// 		resp = fsm.applyDelete(entry.Key)

// 	default:
// 		resp = fmt.Errorf("invalid operation: %x", op)

// 	}

// 	return resp
// }

// func (fsm *BadgerKeyValueFSM) applySet(entry *hexatype.Entry) error {
// 	value := entry.Data[1:]
// 	kv := &KeyValuePair{Entry: entry, Value: value, Key: entry.Key}
// 	val, err := proto.Marshal(kv)
// 	if err == nil {
// 		err = fsm.kv.Set(entry.Key, val, 0)
// 	}

// 	return err
// }

// func (fsm *BadgerKeyValueFSM) applyDelete(key []byte) error {
// 	return fsm.kv.Delete(key)
// }

// // Close closes the underlying badger store
// func (fsm *BadgerKeyValueFSM) Close() error {
// 	return fsm.kv.Close()
// }
