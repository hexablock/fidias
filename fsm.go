package fidias

import (
	"bytes"
	"fmt"

	kelips "github.com/hexablock/go-kelips"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/log"
)

const (
	// OpSet is the op to set a ke-value pair
	opKVSet byte = iota + 1
	// OpDel is the op to delete a key-value pair
	opKVDel
)

// KVStore is the kv store used by the FSM to perform write operations
type KVStore interface {
	// Get a key
	Get(key []byte) (*KVPair, error)

	// Set a key.  Called by the fsm.  It returns any directory keys that may
	// have been implicitly created
	Set(kvp *KVPair) ([]*KVPair, error)

	// Delete a key.  Called by the fsm
	Remove(key []byte) error

	// Iterate over kv's starting at the prefix.  If recurse is true then all
	// keys in subdirs are also returned
	Iter(prefix []byte, recurse bool, f func(kv *KVPair) bool)
}

// FSM is a hexalog FSM for an in-memory key-value store.  It implements the
// FSM interface and provides a get function to retrieve keys as all write
// are handled by the FSM
type FSM struct {
	// Hexalog entry prefix for kv's
	kvprefix []byte

	// Local host DHT address
	localTuple kelips.TupleHost

	// Actual kvstore
	kvs KVStore

	// DHT
	dht DHT
}

// NewFSM inits a new FSM. localTuple is the local host port tuple for the dht
func NewFSM(kvprefix string, localTuple kelips.TupleHost, kvs KVStore) *FSM {
	return &FSM{
		kvprefix:   []byte(kvprefix),
		localTuple: localTuple,
		kvs:        kvs,
	}
}

// RegisterDHT registers the dht to the state machine.  THis is to allow inserts
// to the dht when keys log entries are applied
func (fsm *FSM) RegisterDHT(dht DHT) {
	fsm.dht = dht
}

// Apply applies the given entry to the FSM.  entryID is the hash id of the
// entry.  The first byte in entry.Data contains the operation to be performed
// followed by the actual value.
func (fsm *FSM) Apply(entryID []byte, entry *hexalog.Entry) interface{} {
	if entry.Data == nil || len(entry.Data) == 0 {
		log.Printf("[WARNING] Hexalog entry has no data key=%s", entry.Key)
		return nil
	}

	var (
		op   = entry.Data[0]
		resp interface{}
	)

	switch op {
	case opKVSet:
		resp = fsm.applyKVSet(entryID, entry, entry.Data[1:])

	case opKVDel:
		resp = fsm.applyKVDelete(entry)

	default:
		resp = fmt.Errorf("invalid operation: %x", op)

	}

	return resp
}

func (fsm *FSM) applyKVSet(entryID []byte, entry *hexalog.Entry, value []byte) error {
	kv := &KVPair{
		Key:          bytes.TrimPrefix(entry.Key, fsm.kvprefix),
		Value:        value,
		Modification: entryID,
		ModTime:      entry.Timestamp,
		LTime:        entry.LTime,
		Height:       entry.Height,
	}

	createdDirs, err := fsm.kvs.Set(kv)
	if err != nil {
		return err
	}

	// Insert key to dht
	if err = fsm.dht.Insert(entry.Key, fsm.localTuple); err != nil {
		log.Println("[ERROR] FSM dht insert failed:", err)
	}

	// Insert any directories created to dht
	for _, c := range createdDirs {
		nskey := append(fsm.kvprefix, c.Key...)
		if er := fsm.dht.Insert(nskey, fsm.localTuple); er != nil {
			log.Println("[ERROR] FSM dht insert failed:", er)
			err = er
		}
	}

	log.Printf("[DEBUG] FSM nskey=%s dirs-created=%d height=%d error='%v'",
		entry.Key, len(createdDirs), kv.Height, err)

	return err
}

// ApplyDelete applies a hexalog delete operation entry to the fsm
func (fsm *FSM) applyKVDelete(entry *hexalog.Entry) error {
	key := bytes.TrimPrefix(entry.Key, fsm.kvprefix)
	err := fsm.kvs.Remove(key)
	if err == nil {
		err = fsm.dht.Delete(entry.Key)
	}
	return err
}
