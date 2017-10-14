package fidias

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexatype"
)

type InMemKeyValueFSM struct {
	prefix []byte // Hexalog entry key prefix
	mu     sync.RWMutex
	kv     map[string]*KeyValuePair
}

func NewInMemKeyValueFSM(prefix string) *InMemKeyValueFSM {
	return &InMemKeyValueFSM{
		prefix: []byte(prefix),
		kv:     make(map[string]*KeyValuePair),
	}
}

func (fsm *InMemKeyValueFSM) Get(key []byte) (*KeyValuePair, error) {
	fsm.mu.RLock()
	value, ok := fsm.kv[string(key)]
	if ok {
		fsm.mu.RUnlock()
		return value, nil
	}
	fsm.mu.RUnlock()

	return nil, hexatype.ErrKeyNotFound
}

func (fsm *InMemKeyValueFSM) ApplySet(entryID []byte, entry *hexalog.Entry, value []byte) error {
	key := bytes.TrimPrefix(entry.Key, fsm.prefix)
	kv := &KeyValuePair{Entry: entry, Value: value, Key: key}

	fsm.mu.Lock()
	fsm.kv[string(key)] = kv
	fsm.mu.Unlock()

	return nil
}

func (fsm *InMemKeyValueFSM) ApplyDelete(entry *hexalog.Entry) error {
	key := string(bytes.TrimPrefix(entry.Key, fsm.prefix))

	fsm.mu.RLock()
	if _, ok := fsm.kv[key]; !ok {
		fsm.mu.RUnlock()
		return fmt.Errorf("key not found: %s", key)
	}
	fsm.mu.RUnlock()

	fsm.mu.Lock()
	delete(fsm.kv, key)
	fsm.mu.Unlock()

	return nil
}
