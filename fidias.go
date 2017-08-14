package fidias

import (
	"io"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
)

// KeyValueFSM is an FSM for a key value store.  Aside from fsm functions, it also
// contains read-only key-value functions needed.
type KeyValueFSM interface {
	hexalog.FSM
	Get(key []byte) (*hexatype.KeyValuePair, error)
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
	RebalanceBufSize int           // Rebalance request buffer size
	Replicas         int           // Number of replicas for a key
	StableThreshold  time.Duration // Threshold after ring event to consider we are stable
}

// Hostname returns the configured hostname. The assumption here is the log and ring
// hostnames are the same as they should be checked and set prior to using this call
func (conf *Config) Hostname() string {
	return conf.Ring.Hostname
}

// DefaultConfig returns a default sane config setting the hostname on the log and ring
// configs
func DefaultConfig(hostname string) *Config {
	cfg := &Config{
		Replicas:         3,
		RebalanceBufSize: 32,
		Ring:             hexaring.DefaultConfig(hostname),
		Hexalog:          hexalog.DefaultConfig(hostname),
		StableThreshold:  5 * time.Minute,
	}

	return cfg
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
func New(conf *Config, appFSM KeyValueFSM, logStore *hexalog.LogStore, stableStore hexalog.StableStore, server *grpc.Server) (g *Fidias, err error) {
	g = &Fidias{
		conf:        conf,
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

	kvremote := NewNetTransport(fsm, 30*time.Second, conf.Ring.MaxConnIdle)
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

// NewEntry returns a new Entry for the given key from Hexalog. r is the number of replicas
// to use.  If r is <=0 r is set to the default configured replication count.  It returns
// an error if the node is not part of the location set or a lookup error occurs
func (fidias *Fidias) NewEntry(key []byte, r int) (*hexatype.Entry, *ReMeta, error) {
	// Lookup locations for this key
	locs, err := fidias.ring.LookupReplicated(key, r)
	if err != nil {
		return nil, nil, err
	}

	meta := &ReMeta{PeerSet: locs}

	//
	// TODO: Optimize ???
	//

	if _, err = locs.GetByHost(fidias.conf.Hostname()); err != nil {
		return nil, meta, err
	}

	entry := fidias.hexlog.New(key)
	return entry, meta, nil
}

// ProposeEntry finds locations for the entry and submits a new proposal to those
// locations.
func (fidias *Fidias) ProposeEntry(entry *hexatype.Entry, opts *hexatype.RequestOptions) (*hexalog.Ballot, error) {
	return fidias.hexlog.Propose(entry, opts)
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

	_, err = fidias.ring.ScourReplica(locs[0].ID, func(vn *chord.Vnode) error {
		if k, e := fidias.trans.GetKey(vn.Host, key); e == nil {
			kvp = k
			meta.Vnode = vn
			return io.EOF
		}
		return nil
	})

	if err == io.EOF {
		// We found the entry.
		err = nil
	} else if kvp == nil {
		err = hexatype.ErrKeyNotFound
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
