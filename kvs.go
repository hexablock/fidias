package fidias

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"time"

	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
	"github.com/hexablock/phi"

	"github.com/hexablock/hexalog"
)

// WriteStats contains stats regarding a write operation to the log
// type WriteStats struct {
// 	BallotTime   time.Duration
// 	ApplyTime    time.Duration
// 	Participants []*hexalog.Participant
// }

// ReadStats contains reod operation stats
// type ReadStats struct {
// 	// Node serving the read
// 	Nodes []*hexatype.Node
// 	// Affinity group
// 	Group int
// 	// Node priority in the group
// 	Priority int
// 	// Response time
// 	RespTime time.Duration
// }

// NewKVPair inits a new kv pair with the key and value.
func NewKVPair(key, value []byte) *KVPair {
	return &KVPair{Key: key, Value: value}
}

// IsDir returns if the KVPair is a directory
func (kvp *KVPair) IsDir() bool {
	return os.FileMode(kvp.Flags) == os.ModeDir
}

// MarshalJSON custom marshals the key-value pair to handle hash id's
func (kvp KVPair) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Key          string
		Value        []byte
		Flags        int64
		ModTime      time.Time
		LTime        uint64
		Modification string
		Height       uint32
	}{
		string(kvp.Key),
		kvp.Value,
		kvp.Flags,
		time.Unix(0, int64(kvp.ModTime)),
		kvp.LTime,
		hex.EncodeToString(kvp.Modification),
		kvp.Height,
	})
}

// ReadOptions contains request options for read requests
type ReadOptions struct{}

// WriteOptions contains options to perform write operation
// type WriteOptions struct {
// 	WaitBallot       bool
// 	WaitApply        bool
// 	WaitApplyTimeout time.Duration
// 	Retries          int
// 	RetryInterval    time.Duration
// }

// DefaultWriteOptions returns a sane set of WriteOptions defaults
func DefaultWriteOptions() *WriteOptions {
	return &WriteOptions{
		WaitBallot:       true,
		WaitApply:        true,
		WaitApplyTimeout: (2 * time.Second).Nanoseconds(),
		Retries:          1,
		RetryInterval:    (35 * time.Millisecond).Nanoseconds(),
	}
}

// KVTransport implements a transport to perfor key-value rpc's
type KVTransport interface {
	GetKey(ctx context.Context, host string, key []byte) (*KVPair, error)
	ListDir(ctx context.Context, host string, dir []byte) ([]*KVPair, error)
	Register(kv KVStore)
}

// KVS is a consistent key value store.  It performs write operations by writing
// entries to hexalog which are applied by the FSM to the KVStore
type KVS struct {
	// Prefix to use for log entries.  This allows to serve multiple namespaces
	// with the same logic
	prefix []byte

	// Transport for read requests
	trans KVTransport

	// Log for write requests
	hxl phi.WAL

	// DHT used for lookups to perform gets
	dht phi.DHT
}

// NewKVS inits a new KVS instance using the store for reads and write
// operations by appending entries to the log
//func NewKVS(host, prefix string, kvstore KVStore, wal WAL, remote KVTransport, dht DHT) *KVS {
func NewKVS(prefix string, wal phi.WAL, trans KVTransport, dht phi.DHT) *KVS {
	kv := &KVS{
		prefix: []byte(prefix),
		hxl:    wal,
		trans:  trans,
		dht:    dht,
	}

	return kv
}

// Get returns a KVPair for the key if it exists otherwise an error is returned
func (kvs *KVS) Get(key []byte, opt *ReadOptions) (*KVPair, *ReadStats, error) {
	nskey := append(kvs.prefix, key...)

	start := time.Now()

	nodes, err := kvs.dht.Lookup(nskey)
	if err != nil {
		return nil, nil, err
	}

	if nodes == nil || len(nodes) == 0 {
		return nil, nil, hexatype.ErrKeyNotFound
	}

	var (
		stats = &ReadStats{}
		kvp   *KVPair
	)

	for i, n := range nodes {
		meta := n.Metadata()

		kvp, err = kvs.trans.GetKey(context.Background(), meta["hexalog"], key)
		if err == nil {
			// Set the node returning the response to the first one in the list
			// if it isn't
			if i != 0 {
				nodes[0], nodes[i] = nodes[i], nodes[0]
			}
			stats.Nodes = nodes
			stats.Priority = int32(i)
			stats.RespTime = time.Since(start).Nanoseconds()
			return kvp, stats, nil
		}
	}
	// Set the nodes queried
	stats.Nodes = nodes
	stats.RespTime = time.Since(start).Nanoseconds()

	return nil, stats, err
}

// List performs a lookup on dir and retrieves all children from each node.
func (kvs *KVS) List(dir []byte, opt *ReadOptions) ([]*KVPair, *ReadStats, error) {
	nsdir := append(kvs.prefix, dir...)

	start := time.Now()

	nodes, err := kvs.dht.Lookup(nsdir)
	if err != nil {
		return nil, nil, err
	}

	d := dir
	if dir[len(dir)-1] != '/' {
		d = append(dir, byte('/'))
	}

	out := make(map[string]*KVPair)
	stats := &ReadStats{Nodes: nodes}

	for _, n := range nodes {
		meta := n.Metadata()
		// TODO: Opmitize by selecting the right nodes
		ls, er := kvs.trans.ListDir(context.Background(), meta["hexalog"], d)
		if er != nil {
			err = er
			continue
		}

		for _, l := range ls {
			k := string(l.Key)

			hav, ok := out[k]
			if !ok {
				out[k] = l
				continue
			}

			if l.Height == hav.Height {
				continue
			}
			log.Println("[ERROR] Conflict", l, hav)
		}
	}

	o := make([]*KVPair, 0, len(out))
	for _, v := range out {
		o = append(o, v)
	}

	stats.RespTime = time.Since(start).Nanoseconds()

	return o, stats, err
}

// Set consistently sets a key-value pair by submitting the operation to the log
func (kvs *KVS) Set(kv *KVPair, wo *WriteOptions) (*KVPair, *phi.WriteStats, error) {
	nskey := append(kvs.prefix, kv.Key...)

	var stats *phi.WriteStats

	ent, peers, err := kvs.hxl.NewEntry(nskey)
	if err == nil {
		ent.Data = append([]byte{opKVSet}, kv.Value...)
		opt := buildLogOpts(peers, wo)
		if kv.Modification, stats, err = kvs.hxl.ProposeEntry(ent, opt, int(wo.Retries), time.Duration(wo.RetryInterval)); err == nil {
			kv.Height = ent.Height
			kv.ModTime = ent.Timestamp
			kv.LTime = ent.LTime
			return kv, stats, nil
		}

	}

	return nil, stats, err
}

// CASet checks and sets a key value pair.  mod is the hash id of the last entry
// used as the appension point.  An error is returned if there is a mismatch
// otherwise a KVPair with the new Modification and Height is returned
func (kvs *KVS) CASet(kv *KVPair, mod []byte, wo *WriteOptions) (*KVPair, *phi.WriteStats, error) {
	nskey := append(kvs.prefix, kv.Key...)

	last, err := kvs.hxl.GetEntry(nskey, mod)
	if err != nil {
		return nil, nil, err
	}

	var stats *phi.WriteStats

	ent, peers, err := kvs.hxl.NewEntryFrom(last)
	if err == nil {
		ent.Data = append([]byte{opKVSet}, kv.Value...)
		opt := buildLogOpts(peers, wo)

		// Set retries to 1 as the log may be well ahead
		if kv.Modification, stats, err = kvs.hxl.ProposeEntry(ent, opt, 1, time.Duration(wo.RetryInterval)); err == nil {
			kv.Height = ent.Height
			return kv, stats, nil
		}

	}

	return nil, stats, err
}

// Remove consistently removes a key by submitting a remove operation to the
// log to be applied by the FSM
func (kvs *KVS) Remove(key []byte, wo *WriteOptions) (*phi.WriteStats, error) {
	nskey := append(kvs.prefix, key...)

	var stats *phi.WriteStats

	ent, peers, err := kvs.hxl.NewEntry(nskey)
	if err == nil {
		ent.Data = []byte{opKVDel}
		opt := buildLogOpts(peers, wo)
		_, stats, err = kvs.hxl.ProposeEntry(ent, opt, int(wo.Retries), time.Duration(wo.RetryInterval))
	}

	return stats, err
}

// CARemove checks the mod hash against the last entry and applies the remove.
// It returns an error if there is a mismatch
func (kvs *KVS) CARemove(key []byte, mod []byte, wo *WriteOptions) (*phi.WriteStats, error) {
	nskey := append(kvs.prefix, key...)

	last, err := kvs.hxl.GetEntry(nskey, mod)
	if err != nil {
		return nil, err
	}

	var stats *phi.WriteStats

	ent, peers, err := kvs.hxl.NewEntryFrom(last)
	if err == nil {
		ent.Data = []byte{opKVDel}
		opt := buildLogOpts(peers, wo)
		_, stats, err = kvs.hxl.ProposeEntry(ent, opt, int(wo.Retries), time.Duration(wo.RetryInterval))
	}

	return stats, err
}

// build hexalog request options
func buildLogOpts(peers []*hexalog.Participant, opt *WriteOptions) *hexalog.RequestOptions {
	wo := opt
	if wo == nil {
		wo = DefaultWriteOptions()
	}

	o := hexalog.DefaultRequestOptions()
	o.PeerSet = peers
	o.WaitBallot = wo.WaitBallot
	o.WaitApply = wo.WaitApply
	o.SourceIndex = -1
	o.WaitApplyTimeout = int32(wo.WaitApplyTimeout / 1000000)
	return o
}
