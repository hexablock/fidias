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

const (
	// OpFsSet is used to set a path in the fsm
	OpFsSet byte = iota + 10
	// OpFsDel is used to delete a path from the fsm
	OpFsDel
)

// FileSystemFSM implements an FSM to manage a versioned file-system.
// It is responsible for applying log entries to provide a
// VersionedFile file-system view.
type FileSystemFSM interface {
	// Get a VersionedFile by name
	Get(name string) (*VersionedFile, error)
	// ApplySet is called when an entry needs to be applied. It is called
	// with the entry and the extracted value from the entry. It should
	// use the value bytes as the data payload
	ApplySet(entryID []byte, entry *hexalog.Entry, value []byte) error
	// ApplyDelete is called when a delete entry to needs to be applied
	// It should remove the key and all versions given by the entry key
	ApplyDelete(entry *hexalog.Entry) error
}

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
	kv KeyValueFSM   // FSM for key-value pairs
	fs FileSystemFSM // FSM for filesystem
}

// NewFSM inits a new FSM
func NewFSM(kvprefix, fsprefix string) *FSM {
	return &FSM{
		kv: NewInMemKeyValueFSM(kvprefix),
		fs: NewInMemVersionedFileFSM(fsprefix),
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

// GetPath returns a path with pointers to all of its versions
func (fsm *FSM) GetPath(name string) (*VersionedFile, error) {
	return fsm.fs.Get(name)
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

	case OpFsSet:
		resp = fsm.fs.ApplySet(entryID, entry, entry.Data[1:])

	case OpFsDel:
		resp = fsm.fs.ApplyDelete(entry)

	default:
		resp = fmt.Errorf("invalid operation: %x", op)

	}

	return resp
}

// Close is a no-op to satisfy the KeyValueFSM interface
func (fsm *FSM) Close() error {
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
// func (fsm *BadgerKeyValueFSM) Apply(entryID []byte, entry *hexalog.Entry) interface{} {
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

// func (fsm *BadgerKeyValueFSM) applySet(entry *hexalog.Entry) error {
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
