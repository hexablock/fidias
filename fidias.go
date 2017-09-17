package fidias

import (
	"log"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
)

// KeyValueFSM is an FSM for a key value store.  Aside from fsm functions, it also
// contains read-only key-value functions needed.
type KeyValueFSM interface {
	hexalog.FSM
	Open() error
	GetKey(key []byte) (*KeyValuePair, error)
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
	// Underlying ring
	ring *hexaring.Ring
	// Ring backed hexalog
	hexlog *Hexalog
	// Key-value interface
	keyvs *Keyvs
	// Ring backed BlockDevice
	dev *RingDevice
	// Blocks of keys this node is responsible for. These are the local vnodes and
	// their respective predecessors
	keyblocks *keyBlockSet
	// Relocation engine to send keys to be relocated
	rel *Relocator
	// Fetcher used for log entry fetching
	fet *Fetcher
	// Channel to signal shutdown
	shutdown chan struct{}
}

// New instantiates a new instance of Fidias based on the given config and stores along with
// a grpc server instance to register the network transports
func New(conf *Config, hexlog *Hexalog, relocator *Relocator, fetcher *Fetcher, keyvs *Keyvs, dev *RingDevice, trans *NetTransport) *Fidias {

	fids := &Fidias{
		conf:      conf,
		hexlog:    hexlog,
		keyvs:     keyvs,
		dev:       dev,
		keyblocks: newKeyBlockSet(),
		rel:       relocator,
		fet:       fetcher,
		shutdown:  make(chan struct{}, 1), // For relocator
	}

	// Register hexalog network transport to fetcher
	fids.fet.RegisterTransport(hexlog.trans.remote)
	fids.fet.RegisterHealer(hexlog)
	// Register fetch channels to fidias network transport
	trans.Register(fetcher.fetCh, fetcher.blkCh)

	// Register fidias transport to relocator
	fids.rel.RegisterTransport(trans)

	// Register keyvs transport
	fids.keyvs.RegisterTransport(trans)

	// Set self as the chord delegate
	conf.Ring.Delegate = fids

	return fids
}

// Register registers the chord ring to fidias.  This is due to the fact that guac and the
// ring depend on each other and the ring may not be intialized yet.  Only upon ring
// registration, the rebalancing is started.
func (fidias *Fidias) Register(ring *hexaring.Ring) {
	fidias.ring = ring

	// Register dht to hexalog
	fidias.hexlog.Register(ring)

	// Register to key-value
	fidias.keyvs.RegisterLocator(ring)

	// Register dht to storage device if enabled
	if fidias.dev != nil {
		fidias.dev.Register(ring)
	}

	// Register dht to fetcher
	fidias.fet.RegisterLocator(ring)

	log.Println("[INFO] Fidias intializied")
}

func (fidias *Fidias) shutdownWait() {
	fidias.fet.stop()
}
