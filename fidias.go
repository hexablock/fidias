package fidias

import (
	"log"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexaring"
)

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
	// FSM for both keyvs and the file-system
	fsm *FSM
	// Key-value interface
	keyvs *Keyvs
	// Ring backed BlockDevice
	dev *RingDevice
	// Filesystem
	fs *FileSystem

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
func New(conf *Config, hexlog *Hexalog, fsm *FSM, relocator *Relocator, fetcher *Fetcher,
	keyvs *Keyvs, dev *RingDevice, trans *NetTransport) *Fidias {

	fids := &Fidias{
		conf:      conf,
		hexlog:    hexlog,
		fsm:       fsm,
		keyvs:     keyvs,
		dev:       dev,
		keyblocks: newKeyBlockSet(),
		rel:       relocator,
		fet:       fetcher,
		shutdown:  make(chan struct{}, 1), // For relocator
	}

	// Register hexalog network transport to fetcher
	fids.fet.RegisterTransport(hexlog.trans.remote)

	// Register hexalog healer to fetcher
	fids.fet.RegisterHealer(hexlog)

	// Register fetch channels to fidias network transport
	trans.Register(fetcher.fetCh, fetcher.blkCh)

	// Register fidias transport to relocator
	fids.rel.RegisterTransport(trans)

	// Register keyvs transport
	fids.keyvs.RegisterTransport(trans)

	// Init FS
	fids.initFileSystem(trans)

	// Set self as the chord delegate
	conf.Ring.Delegate = fids

	return fids
}

func (fids *Fidias) initFileSystem(trans *NetTransport) {
	if fids.dev == nil {
		return
	}

	fids.fs = NewFileSystem(fids.conf.Hostname(), fids.conf.FileSystemNamespace,
		fids.dev, fids.hexlog, fids.fsm)
	// Register file-system transport
	fids.fs.RegisterTransport(trans)
}

// FileSystem returns the fidias file-system
func (fids *Fidias) FileSystem() *FileSystem {
	return fids.fs
}

// Register registers the chord ring to fidias.  This is due to the fact that guac and the
// ring depend on each other and the ring may not be intialized yet.  Only upon ring
// registration, the rebalancing is started.
func (fids *Fidias) Register(ring *hexaring.Ring) {
	fids.ring = ring

	// Register dht to hexalog
	fids.hexlog.RegisterDHT(ring)
	// Register dht to key-value
	fids.keyvs.RegisterDHT(ring)

	// Register dht to storage device if enabled.  Only init FS is device is initialized
	if fids.dev != nil {

		fids.dev.RegisterDHT(ring)
		fids.fs.RegisterDHT(ring)

		log.Printf("[INFO] FileSystem initialization complete")
	}

	// Register dht to fetcher
	fids.fet.RegisterDHT(ring)

	log.Println("[INFO] Fidias intializied")
}

func (fids *Fidias) shutdownWait() {
	fids.fet.stop()
}
