package fidias

import (
	"time"

	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
	"google.golang.org/grpc"
)

// Config hold the guac config along with the underlying log and ring config
type Config struct {
	Ring             *hexaring.Config
	Hexalog          *hexalog.Config
	RebalanceBufSize int // rebalance request buffer size
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
		RebalanceBufSize: 16,
		Ring:             hexaring.DefaultConfig(hostname),
		Hexalog:          hexalog.DefaultConfig(hostname),
	}
}

// Fidias is the core that manages all operations for a node.  It primary manages
// rebalancing, replication, and appropriately deals with cluster churn.
type Fidias struct {
	conf        *Config
	logstore    hexalog.LogStore       // Key based log store
	logtrans    *hexalog.NetTransport  // Log transport
	hexlog      *hexalog.Hexalog       // Overall log manager
	ring        *hexaring.Ring         // Underlying chord ring
	rebalanceCh chan *RebalanceRequest // Rebalance request channel i.e transfer/takeover
	shutdown    chan struct{}          // Channel to signal shutdown
}

// New instantiates a new instance of Fidias based on the given config
func New(conf *Config, appFSM hexalog.FSM, logStore hexalog.LogStore, stableStore hexalog.StableStore, server *grpc.Server) (g *Fidias, err error) {
	g = &Fidias{
		conf:        conf,
		logstore:    logStore,
		rebalanceCh: make(chan *RebalanceRequest, conf.RebalanceBufSize),
		shutdown:    make(chan struct{}, 1),
	}

	// Set guac as the chord delegate
	conf.Ring.Delegate = g

	// Init hexalog transport and register with gRPC
	g.logtrans = hexalog.NewNetTransport(30*time.Second, conf.Ring.MaxConnIdle)
	hexalog.RegisterHexalogRPCServer(server, g.logtrans)

	// Init hexalog with guac as the FSM
	var fsm hexalog.FSM
	if appFSM == nil {
		fsm = &DummyFSM{}
	} else {
		fsm = appFSM
	}
	g.hexlog, err = hexalog.NewHexalog(conf.Hexalog, fsm, logStore, stableStore, g.logtrans)

	return
}

// Status returns the status of this node
func (fidias *Fidias) Status() interface{} {
	return fidias.ring.Status()
}

// NewEntry returns a new Entry for the given key from Hexalog
func (fidias *Fidias) NewEntry(key []byte) *hexalog.Entry {
	return fidias.hexlog.New(key)
}

// Register registers the chord ring to fidias.  This is due to the fact that guac and the
// ring depend on each other and the ring may not be intialized yet.  Only upon ring
// registration, the rebalancing is started.
func (fidias *Fidias) Register(ring *hexaring.Ring) {
	fidias.ring = ring
	go fidias.startRebalancing()
}

// ProposeEntry finds locations for the entry and submits a new proposal to those
// locations.
func (fidias *Fidias) ProposeEntry(entry *hexalog.Entry) (*hexalog.Ballot, error) {
	// Lookup locations for this key
	locs, err := fidias.ring.LookupReplicated(entry.Key, fidias.conf.Replicas)
	if err != nil {
		return nil, err
	}

	opts := &hexalog.RequestOptions{PeerSet: locs}
	return fidias.hexlog.Propose(entry, opts)
}
