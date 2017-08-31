package fidias

import (
	"io"
	"time"

	"google.golang.org/grpc"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexalog/store"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
)

// KeyValueFSM is an FSM for a key value store.  Aside from fsm functions, it also
// contains read-only key-value functions needed.
type KeyValueFSM interface {
	hexalog.FSM
	Open() error
	Get(key []byte) (*hexatype.KeyValuePair, error)
	Close() error
}

// ReMeta contains metadata associated to a request or response
type ReMeta struct {
	Vnode   *chord.Vnode         // vnode processing the request or response
	PeerSet hexaring.LocationSet // set of peers involved
}

// Fidias is the core that manages all operations for a node.  It primary manages
// rebalancing, replication, and appropriately deals with cluster churn.
type Fidias struct {
	// Configuration
	conf *Config
	// Underlying chord ring
	ring *hexaring.Ring
	// Transport to handle local and remote calls
	trans *localTransport
	// Overall log manager
	hexlog *hexalog.Hexalog
	// FSM
	fsm KeyValueFSM
	// Blocks of keys this node is responsible for. These are the local vnodes and their
	// respective predecessors
	keyblocks *keyBlockSet
	// Relocation engine to send keys to be relocated
	rel *Relocator
	// Fetcher used for log entry fetching
	fet *fetcher
	// Channel to signal shutdown
	shutdown chan struct{}
}

// New instantiates a new instance of Fidias based on the given config and stores along with
// a grpc server instance to register the network transports
func New(conf *Config, appFSM KeyValueFSM, idx store.IndexStore, entries store.EntryStore, logStore *hexalog.LogStore, stableStore hexalog.StableStore, server *grpc.Server) (g *Fidias, err error) {
	// Init the FSM
	var fsm KeyValueFSM
	if appFSM == nil {
		fsm = &DummyFSM{}
	} else {
		fsm = appFSM
	}

	// Open the fsm to begin using it
	if err = fsm.Open(); err != nil {
		return nil, err
	}

	g = &Fidias{
		conf:      conf,
		fsm:       fsm,
		keyblocks: newKeyBlockSet(),
		shutdown:  make(chan struct{}, 1), // relocator
	}
	// Init fidias network transport
	trans := NewNetTransport(fsm, idx, 30*time.Second, conf.Ring.MaxConnIdle, conf.Replicas, conf.Hexalog.Hasher)
	RegisterFidiasRPCServer(server, trans)

	// Init hexalog transport and register with gRPC
	logtrans := hexalog.NewNetTransport(30*time.Second, conf.Ring.MaxConnIdle)
	hexalog.RegisterHexalogRPCServer(server, logtrans)

	// Set self as the chord delegate
	conf.Ring.Delegate = g

	g.trans = &localTransport{
		host:    conf.Hostname(),
		local:   logStore,
		remote:  logtrans,
		kvlocal: fsm,
		ftrans:  trans,
	}

	err = g.initHexalog(fsm, idx, entries, stableStore)
	return
}

func (fidias *Fidias) initHexalog(fsm KeyValueFSM, idx store.IndexStore, entries store.EntryStore, stable hexalog.StableStore) (err error) {
	tr := fidias.trans
	c := fidias.conf
	fidias.rel = NewRelocator(fidias.conf, idx, tr.ftrans)
	fidias.hexlog, err = hexalog.NewHexalog(c.Hexalog, fsm, tr.local, stable, tr.remote)
	if err == nil {
		// setup log entry fetcher
		fidias.fet = newFetcher(c, idx, entries, fidias.hexlog, tr)
		// register fetch channel
		tr.ftrans.Register(fidias.fet.fetCh)
	}

	return err
}

// Register registers the chord ring to fidias.  This is due to the fact that guac and the
// ring depend on each other and the ring may not be intialized yet.  Only upon ring
// registration, the rebalancing is started.
func (fidias *Fidias) Register(ring *hexaring.Ring) {
	fidias.ring = ring
	fidias.fet.register(ring)
}

// NewEntry returns a new Entry for the given key from Hexalog.  It returns an error if
// the node is not part of the location set or a lookup error occurs
func (fidias *Fidias) NewEntry(key []byte) (*hexatype.Entry, *ReMeta, error) {

	// Lookup locations for this key
	locs, err := fidias.ring.LookupReplicated(key, fidias.conf.Hexalog.Votes)
	if err != nil {
		return nil, nil, err
	}

	meta := &ReMeta{PeerSet: locs}

	//
	// TODO: Optimize ???
	//

	//var self *hexaring.Location
	if _, err = locs.GetByHost(fidias.conf.Hostname()); err != nil {
		return nil, meta, err
	}

	entry := fidias.hexlog.New(key)
	return entry, meta, nil
}

// ProposeEntry finds locations for the entry and submits a new proposal to those
// locations.
func (fidias *Fidias) ProposeEntry(entry *hexatype.Entry, opts *hexatype.RequestOptions) (ballot *hexalog.Ballot, err error) {
	retries := int(opts.Retries)
	if retries < 1 {
		retries = 1
	}

	for i := 0; i < retries; i++ {
		// Propose with retries.  Retry only if it is a ErrPreviousHash error
		if ballot, err = fidias.hexlog.Propose(entry, opts); err == nil {
			return
		} else if err == hexatype.ErrPreviousHash {
			time.Sleep(fidias.conf.RetryInterval)
		} else {
			return
		}

	}

	return
}

// GetEntry tries to get an entry from the ring.  It gets the replica locations and queries
// upto the max allowed successors for each location.
func (fidias *Fidias) GetEntry(key, id []byte) (entry *hexatype.Entry, meta *ReMeta, err error) {
	meta = &ReMeta{}
	_, err = fidias.ring.ScourReplicatedKey(key, fidias.conf.Replicas, func(vn *chord.Vnode) error {
		ent, er := fidias.trans.GetEntry(vn.Host, key, id)
		if er == nil {
			entry = ent
			meta.Vnode = vn
			return io.EOF
		}

		return nil
	})

	// We found the entry.
	if err == io.EOF {
		err = nil
	} else if entry == nil {
		err = hexatype.ErrEntryNotFound
	}

	return
}

// GetKey tries to get a key-value pair from a given replica set on the ring.  This is not
// be confused with the log key.  It scours the first replica only.
func (fidias *Fidias) GetKey(key []byte) (kvp *hexatype.KeyValuePair, meta *ReMeta, err error) {

	locs, err := fidias.ring.LookupReplicated(key, fidias.conf.Replicas)
	if err != nil {
		return nil, nil, err
	}
	meta = &ReMeta{PeerSet: locs}

	// Scour the leader replica range
	_, err = fidias.ring.ScourReplica(locs[0].ID, func(vn *chord.Vnode) error {
		if k, e := fidias.trans.GetKey(vn.Host, key); e == nil {
			kvp = k
			meta.Vnode = vn
			return io.EOF
		}
		return nil
	})

	// We found the entry.
	if err == io.EOF {
		err = nil
	} else if kvp == nil {
		err = hexatype.ErrKeyNotFound
	}

	return
}

// Leader returns the leader of the given location set from the underlying log.
func (fidias *Fidias) Leader(key []byte, locs hexaring.LocationSet) (*hexalog.KeyLeader, error) {
	return fidias.hexlog.Leader(key, locs)
}

func (fidias *Fidias) shutdownWait() {
	fidias.fet.stop()
}
