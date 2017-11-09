package fidias

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

// InmemKVStore implements a in-memory key-value store used by the FSM
type InmemKVStore struct {
	mu sync.RWMutex
	kv map[string]*KVPair
}

// NewInmemKVStore implements in in-memory kv store using a map
func NewInmemKVStore() *InmemKVStore {
	return &InmemKVStore{
		kv: make(map[string]*KVPair),
	}
}

// Get returns a KVPair for the given key
func (kvs *InmemKVStore) Get(key []byte) (*KVPair, error) {
	k := string(key)

	kvs.mu.RLock()
	value, ok := kvs.kv[k]
	if !ok {
		kvs.mu.RUnlock()
		return nil, hexatype.ErrKeyNotFound
	}
	kvs.mu.RUnlock()

	return value, nil

}

// Iter iterates over each key matching the prefix.  If the callback returns
// false iteration is immediately terminated
func (kvs *InmemKVStore) Iter(prefix []byte, recurse bool, f func(kvp *KVPair) bool) {
	pre := string(prefix)

	kvs.mu.RLock()

	keys := kvs.sortedKeys()
	var seek int
	for i, k := range keys {
		if strings.HasPrefix(k, pre) {
			seek = i
			break
		}
	}

	if recurse {

		for _, k := range keys[seek:] {
			// Break as we've passed the prefix
			if !strings.HasPrefix(k, pre) {
				break
			}
			if !f(kvs.kv[k]) {
				break
			}
		}

	} else {

		for _, k := range keys[seek:] {
			// Break as we've passed the prefix
			if !strings.HasPrefix(k, pre) {
				break
			}

			// Trim prefix to check if it's a dir
			sk := strings.TrimPrefix(k, pre)
			if strings.Contains(sk, "/") {
				continue
			}

			if !f(kvs.kv[k]) {
				break
			}

		}

	}

	kvs.mu.RUnlock()
}

func (kvs *InmemKVStore) sortedKeys() []string {
	keys := make([]string, 0, len(kvs.kv))
	for k := range kvs.kv {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	return keys
}

// Set writes the KVPair to the store.  This is meant to be directly called only
// by the fsm to ensure consistency.  It returns any directories created as part
// of writing out the given key
func (kvs *InmemKVStore) Set(kvp *KVPair) ([]*KVPair, error) {
	k := string(kvp.Key)

	kvs.mu.RLock()
	if val, ok := kvs.kv[k]; ok {
		if val.Flags != kvp.Flags {
			kvs.mu.RUnlock()
			return nil, fmt.Errorf("cannot change key-value type")
		}
	}
	kvs.mu.RUnlock()

	kvs.mu.Lock()
	// Assign key
	kvs.kv[k] = kvp
	// Create any required dirs
	created := kvs.upsertPathDir(kvp)
	kvs.mu.Unlock()

	return created, nil
}

// Remove removes a key from the store.  This is meant to be directly called only
// by the fsm to ensure consistency
func (kvs *InmemKVStore) Remove(key []byte) error {
	k := string(key)

	kvs.mu.Lock()
	defer kvs.mu.Unlock()

	if _, ok := kvs.kv[k]; ok {
		delete(kvs.kv, k)
		return nil
	}

	return hexatype.ErrKeyNotFound
}

func (kvs *InmemKVStore) upsertPathDir(kvp *KVPair) []*KVPair {
	key := kvp.Key

	created := make([]*KVPair, 0)

	for i, c := range key {
		if c == '/' {
			k := string(key[:i])
			if _, ok := kvs.kv[k]; ok {
				continue
			}

			kv := NewKVPair(nil, nil)
			kv.Key = make([]byte, len(key[:i]))
			copy(kv.Key, key[:i])

			kv.Flags = int64(os.ModeDir)
			kv.Modification = kvp.Modification
			kv.Height = kvp.Height
			kv.ModTime = kvp.ModTime
			kv.LTime = kvp.LTime

			log.Printf("[DEBUG] Created dir=%s", kv.Key)
			created = append(created, kv)
			kvs.kv[k] = kv
		}
	}

	return created
}
