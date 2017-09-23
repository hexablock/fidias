package fidias

import (
	"bytes"
	"sync"

	"github.com/hexablock/hexatype"
)

type InMemVersionedFileFSM struct {
	prefix []byte // hexalog entry key prefix
	mu     sync.RWMutex
	fs     map[string]*VersionedFile
}

func NewInMemVersionedFileFSM(prefix string) *InMemVersionedFileFSM {
	return &InMemVersionedFileFSM{
		prefix: []byte(prefix),
		fs:     make(map[string]*VersionedFile),
	}
}

func (store *InMemVersionedFileFSM) Get(name string) (*VersionedFile, error) {
	store.mu.RLock()
	ver, ok := store.fs[name]
	if !ok {
		store.mu.RUnlock()
		return nil, errFileNotFound
	}

	store.mu.RUnlock()
	return ver, nil
}

func (store *InMemVersionedFileFSM) ApplySet(entryID []byte, entry *hexatype.Entry, value []byte) error {
	key := bytes.TrimPrefix(entry.Key, store.prefix)
	ver := NewVersionedFile(string(key))

	err := ver.UnmarshalBinary(value)
	if err != nil {
		return err
	}

	store.mu.Lock()
	store.fs[ver.name] = ver
	store.mu.Unlock()

	return nil
}

// ApplyRemove
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
