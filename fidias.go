package fidias

import (
	"log"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
)

// KeyValueFSM is an FSM for a key value store.  Aside from fsm functions, it also
// contains read-only key-value functions needed.
type KeyValueFSM interface {
	hexalog.FSM
	Open() error
	GetKey(key []byte) (*hexatype.KeyValuePair, error)
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

	hexlog *Hexalog

	// Key-value interface
	keyvs *Keyvs

	// Blox device
	dev *RingDevice

	// Transport to handle local and remote calls
	//ftrans *localTransport
	// Overall log manager
	//hexlog *hexalog.Hexalog

	// FSM
	//fsm KeyValueFSM

	// Blocks of keys this node is responsible for. These are the local vnodes and their
	// respective predecessors
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
	//func New(conf *Config, appFSM KeyValueFSM, idx store.IndexStore, entries store.EntryStore, logStore *hexalog.LogStore, stableStore hexalog.StableStore, server *grpc.Server) (g *Fidias, err error) {

	// Init the FSM
	// var fsm KeyValueFSM
	// if appFSM == nil {
	// 	fsm = &DummyFSM{}
	// } else {
	// 	fsm = appFSM
	// }

	// Open the fsm to begin using it
	// if err = fsm.Open(); err != nil {
	// 	return nil, err
	// }

	// Init fidias network transport
	//reapInt := 30 * time.Second
	//maxIdle := conf.Ring.MaxConnIdle
	//trans := NewNetTransport(fsm, idx, reapInt, maxIdle, conf.Hexalog.Votes, conf.Hexalog.Hasher)
	//RegisterFidiasRPCServer(server, trans)

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

	// Register fidias transport to relocator
	fids.rel.RegisterTransport(trans)
	// Register keyvs transport
	fids.keyvs.RegisterTransport(trans)
	// Register fetch channels to fidias network transport
	trans.Register(fetcher.fetCh, fetcher.blkCh)

	// Init hexalog transport and register with gRPC
	//logtrans := hexalog.NewNetTransport(30*time.Second, conf.Ring.MaxConnIdle)
	//hexalog.RegisterHexalogRPCServer(server, logtrans)

	// Block stuff
	// netopt:=blox.DefaultNetClientOptions(conf.Hasher())
	// bremote:=blox.NewNetTransport(ln, netopt)
	// bloxTrans:=blox.NewLocalTransport(host, bremote.NetClient)
	//rdev, err := device.NewFileRawDevice()
	//blkdev := device.NewBlockDevice(rdev)
	// bloxTrans.Register(blkdev)
	//dev:=NewRingDevice(replicas, hasher, bloxTrans)

	// Set self as the chord delegate
	conf.Ring.Delegate = fids

	// Init local transport
	// g.ftrans = &localTransport{
	// 	host:    conf.Hostname(),
	// 	kvlocal: fsm,
	// 	trans:   trans,
	// }

	//err = g.initHexalog(fsm, idx, entries, logStore, stableStore, logtrans)
	return fids
}

//func (fidias *Fidias) initHexalog(fsm KeyValueFSM, idx store.IndexStore, entries store.EntryStore, logstore *hexalog.LogStore, stable hexalog.StableStore, remote *hexalog.NetTransport) (err error) {
//tr := fidias.trans
// c := fidias.conf
//
// fidias.Hexalog = &Hexalog{
// 	conf:          c.Hexalog,
// 	retryInterval: c.RetryInterval,
// 	trans: &localHexalogTransport{
// 		host:     c.Hostname(),
// 		logstore: logstore,
// 		remote:   remote,
// 	},
// }
//
// fidias.Hexalog.hexlog, err = hexalog.NewHexalog(c.Hexalog, fsm, logstore, stable, remote)
// if err == nil {
// 	// setup log entry fetcher
// 	fidias.fet = newFetcher(c, idx, entries, fidias.Hexalog.hexlog, remote)
// 	// register fetch channel
// 	fidias.ftrans.trans.Register(fidias.fet.fetCh)
// }
//
// return err
//}

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
