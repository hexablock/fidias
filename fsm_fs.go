package fidias

import (
	"bytes"
	"sync"

	"github.com/hexablock/hexatype"
)

// InMemVersionedFileFSM is a VersionedFile FSM
type InMemVersionedFileFSM struct {
	prefix []byte                    // entry key prefix to trim for actual fs path
	mu     sync.RWMutex              // store map lock
	fs     map[string]*VersionedFile // in-memory store
}

// NewInMemVersionedFileFSM inits a new in-memory VersionedFile fsm.  It takes
// a prefix that must be present in hexalog used to trim the received entry
// key to obtain the file path.
func NewInMemVersionedFileFSM(prefix string) *InMemVersionedFileFSM {
	return &InMemVersionedFileFSM{
		prefix: []byte(prefix),
		fs:     make(map[string]*VersionedFile),
	}
}

// Get returns the VersionedFile by the given name
func (store *InMemVersionedFileFSM) Get(name string) (*VersionedFile, error) {
	store.mu.RLock()
	ver, ok := store.fs[name]
	if !ok {
		store.mu.RUnlock()
		return nil, errFileOrDirNotFound
	}
	store.mu.RUnlock()
	return ver, nil
}

// ApplySet applies a set fsm operation for VersionedFiles.  This is not to be directly
// used or called. It is called by the managing parent fsm when a fs set operation
// entry is received by the managing parent fsm.
func (store *InMemVersionedFileFSM) ApplySet(entryID []byte, entry *hexatype.Entry, value []byte) error {
	key := bytes.TrimPrefix(entry.Key, store.prefix)
	ver := NewVersionedFile(string(key))
	ver.entry = entry
	err := ver.UnmarshalBinary(value)
	if err != nil {
		return err
	}

	store.mu.Lock()
	store.fs[ver.name] = ver
	store.mu.Unlock()

	return nil
}

// ApplyDelete applies a delete fsm operation for VersionedFiles.  This is not to be
// directly used or called. It is called by the managing parent fsm when a fs delete
// operation entry is received.
func (store *InMemVersionedFileFSM) ApplyDelete(entry *hexatype.Entry) error {
	key := bytes.TrimPrefix(entry.Key, store.prefix)
	k := string(key)

	store.mu.RLock()
	if _, ok := store.fs[k]; !ok {
		store.mu.RUnlock()
		return hexatype.ErrKeyNotFound
	}
	store.mu.RUnlock()

	store.mu.Lock()
	delete(store.fs, k)
	store.mu.Unlock()

	return nil
}
