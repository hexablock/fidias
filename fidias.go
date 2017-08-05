package fidias

import (
	"io"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
)

// KeyValueFSM is an FSM for a key value store.  Aside from fsm functions, it also
// contains key-value functions needed.
type KeyValueFSM interface {
	hexalog.FSM
	Get(key []byte) (*KeyValuePair, error)
}

// ReMeta contains metadata associated to a request or response
type ReMeta struct {
	Vnode   *chord.Vnode         // vnode processing the request or response
	PeerSet hexaring.LocationSet // set of peers involved
}

// Config hold the guac config along with the underlying log and ring config
type Config struct {
	Ring             *hexaring.Config
	Hexalog          *hexalog.Config
	RebalanceBufSize int // Rebalance request buffer size
	Replicas         int // Number of replicas for a key
}

// Hostname returns the configured hostname. The assumption here is the log and ring
// hostnames are the same as they should be checked and set prior to using this call
func (conf *Config) Hostname() string {
	return conf.Ring.Hostname
}

// DefaultConfig returns a default sane config setting the hostname on the log and ring
// configs
func DefaultConfig(hostname string) *Config {
	return &Config{
		Replicas:         3,
		RebalanceBufSize: 32,
		Ring:             hexaring.DefaultConfig(hostname),
		Hexalog:          hexalog.DefaultConfig(hostname),
	}
}

// Fidias is the core that manages all operations for a node.  It primary manages
// rebalancing, replication, and appropriately deals with cluster churn.
type Fidias struct {
	conf          *Config
	ring          *hexaring.Ring // Underlying chord ring
	tmu           sync.RWMutex   // Ring event time lock
	lastRingEvent time.Time      // Last time there was a ring membership change

	trans *localTransport // Transport to handle local and remote calls

	hexlog *hexalog.Hexalog // Overall log manager

	rebalanceCh chan *RebalanceRequest // Rebalance request channel i.e transfer/takeover
	shutdown    chan struct{}          // Channel to signal shutdown
}

// New instantiates a new instance of Fidias based on the given config
func New(conf *Config, appFSM KeyValueFSM, logStore hexalog.LogStore, stableStore hexalog.StableStore, server *grpc.Server) (g *Fidias, err error) {
	g = &Fidias{
		conf: conf,
		//logstore:    logStore,
		rebalanceCh: make(chan *RebalanceRequest, conf.RebalanceBufSize),
		shutdown:    make(chan struct{}, 2), // healer, rebalancer
	}

	// Set guac as the chord delegate
	conf.Ring.Delegate = g

	// Init hexalog transport and register with gRPC
	logtrans := hexalog.NewNetTransport(30*time.Second, conf.Ring.MaxConnIdle)
	hexalog.RegisterHexalogRPCServer(server, logtrans)

	// Init the FSM
	var fsm KeyValueFSM
	if appFSM == nil {
		fsm = &DummyFSM{}
	} else {
		fsm = appFSM
	}

	kvremote := &NetTransport{kvs: fsm}
	g.trans = &localTransport{
		host:     conf.Hostname(),
		local:    logStore,
		remote:   logtrans,
		kvlocal:  fsm,
		kvremote: kvremote,
	}
	// Register key-value rpc
	RegisterFidiasRPCServer(server, kvremote)

	g.hexlog, err = hexalog.NewHexalog(conf.Hexalog, fsm, logStore, stableStore, logtrans)

	return
}

// Status returns the status of this node
func (fidias *Fidias) Status() interface{} {
	return fidias.ring.Status()
}

// Register registers the chord ring to fidias.  This is due to the fact that guac and the
// ring depend on each other and the ring may not be intialized yet.  Only upon ring
// registration, the rebalancing is started.
func (fidias *Fidias) Register(ring *hexaring.Ring) {
	fidias.ring = ring

	go fidias.startHealer()
	go fidias.startRebalancer()
}

// NewEntry returns a new Entry for the given key from Hexalog
func (fidias *Fidias) NewEntry(key []byte) *hexalog.Entry {
	return fidias.hexlog.New(key)
}

// ProposeEntry finds locations for the entry and submits a new proposal to those
// locations.
func (fidias *Fidias) ProposeEntry(entry *hexalog.Entry) (*hexalog.Ballot, *ReMeta, error) {
	// Lookup locations for this key
	locs, err := fidias.ring.LookupReplicated(entry.Key, fidias.conf.Replicas)
	if err != nil {
		return nil, nil, err
	}

	opts := &hexalog.RequestOptions{PeerSet: locs}
	ballot, err := fidias.hexlog.Propose(entry, opts)
	return ballot, &ReMeta{PeerSet: hexaring.LocationSet(locs)}, err
}

// GetEntry tries to get an entry from the ring.  It gets the replica locations and queries
// upto the max allowed successors for each location.
func (fidias *Fidias) GetEntry(key, id []byte) (entry *hexalog.Entry, meta *ReMeta, err error) {
	meta = &ReMeta{}
	_, err = fidias.ring.Orbit(key, fidias.conf.Replicas, func(vn *chord.Vnode) error {
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
		err = hexalog.ErrEntryNotFound
	}

	return
}

// GetKey tries to get a key-value pair from the ring.  This is not be confused with the
// log key.  It orbits the ring to return the first occurence of the key-value pair.
func (fidias *Fidias) GetKey(key []byte) (kvp *KeyValuePair, meta *ReMeta, err error) {
	meta = &ReMeta{}

	_, err = fidias.ring.Orbit(key, fidias.conf.Replicas, func(vn *chord.Vnode) error {
		kv, er := fidias.trans.GetKey(vn.Host, key)
		if er == nil {
			kvp = kv
			meta.Vnode = vn
			// We found the entry so we return an EOF
			return io.EOF
		}

		return nil
	})

	// We found the entry.
	if err == io.EOF {
		err = nil
	} else if kvp == nil {
		err = errKeyNotFound
	}

	return
}

func (fidias *Fidias) shutdownWait() {
	close(fidias.rebalanceCh)
	// wait for shutdown
	for i := 0; i < 2; i++ {
		<-fidias.shutdown
	}
}
