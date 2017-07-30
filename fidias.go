package fidias

import (
	"log"
	"time"

	"google.golang.org/grpc"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
)

// ReMeta contains metadata associated to a request or response
type ReMeta struct {
	Vnode   *chord.Vnode         // vnode processing the request or response
	PeerSet hexaring.LocationSet // set of peers involved
}

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
		RebalanceBufSize: 32,
		Ring:             hexaring.DefaultConfig(hostname),
		Hexalog:          hexalog.DefaultConfig(hostname),
	}
}

// Fidias is the core that manages all operations for a node.  It primary manages
// rebalancing, replication, and appropriately deals with cluster churn.
type Fidias struct {
	conf        *Config
	ring        *hexaring.Ring // Underlying chord ring
	trans       *localTransport
	logstore    hexalog.LogStore       // Key based log store
	logtrans    *hexalog.NetTransport  // Log transport
	hexlog      *hexalog.Hexalog       // Overall log manager
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

	g.trans = &localTransport{host: conf.Hostname(), local: logStore, remote: g.logtrans}

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

// Register registers the chord ring to fidias.  This is due to the fact that guac and the
// ring depend on each other and the ring may not be intialized yet.  Only upon ring
// registration, the rebalancing is started.
func (fidias *Fidias) Register(ring *hexaring.Ring) {
	fidias.ring = ring
	go fidias.start()
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

// GetEntry gets any entry from the ring up to the max number of successors.
func (fidias *Fidias) GetEntry(key, id []byte) (entry *hexalog.Entry, meta *ReMeta, err error) {
	_, vns, err := fidias.ring.Lookup(fidias.conf.Ring.NumSuccessors, key)
	if err != nil {
		return nil, nil, err
	}

	meta = &ReMeta{}
	// Try the primary first
	log.Printf("Trying key=%s id=%x host=%s", key, id, vns[0].Host)
	entry, err = fidias.trans.GetEntry(vns[0].Host, key, id)
	if err == nil {
		meta.Vnode = vns[0]
		return
	}

	tried := map[string]bool{vns[0].Host: true}

	// Try the remaining vnode successors
	for _, vn := range vns[1:] {
		if _, ok := tried[vn.Host]; ok {
			continue
		}
		tried[vn.Host] = true

		log.Printf("Trying key=%s id=%x host=%s", key, id, vn.Host)
		entry, err = fidias.trans.GetEntry(vn.Host, key, id)
		if err == nil {
			meta.Vnode = vn
			return
		}

	}

	err = hexalog.ErrEntryNotFound
	return
}
