package fidias

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"

	"github.com/hashicorp/memberlist"
	"github.com/hexablock/blox"
	"github.com/hexablock/blox/device"
	"github.com/hexablock/go-kelips"
	hexaboltdb "github.com/hexablock/hexa-boltdb"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/vivaldi"
)

// DHT implements a distributed hash table needed to route keys
type DHT interface {
	LookupGroupNodes(key []byte) ([]*hexatype.Node, error)
	Lookup(key []byte) ([]*hexatype.Node, error)
	Insert(key []byte, tuple kelips.TupleHost) error
	Delete(key []byte) error
}

// WALTransport implements an interface for network log operations
type WALTransport interface {
	NewEntry(host string, key []byte, opts *hexalog.RequestOptions) (*hexalog.Entry, error)
	ProposeEntry(ctx context.Context, host string, entry *hexalog.Entry, opts *hexalog.RequestOptions) (*hexalog.ReqResp, error)
	GetEntry(host string, key []byte, id []byte, opts *hexalog.RequestOptions) (*hexalog.Entry, error)
}

// WAL implements an interface to provide p2p distributed consensus
type WAL interface {
	NewEntry(key []byte) (*hexalog.Entry, []*hexalog.Participant, error)
	NewEntryFrom(entry *hexalog.Entry) (*hexalog.Entry, []*hexalog.Participant, error)
	ProposeEntry(entry *hexalog.Entry, opts *hexalog.RequestOptions, retries int, retryInt time.Duration) ([]byte, *WriteStats, error)
	GetEntry(key []byte, id []byte) (*hexalog.Entry, error)
}

// Fidias is core engine for a cluster member/participant it runs a server and
// client components
type Fidias struct {
	conf *Config

	// Local node
	local hexatype.Node

	// Lamport clock
	ltime *hexatype.LamportClock

	// Virtual coorinates
	coord *vivaldi.Client

	// DHT
	dht *kelips.Kelips

	// Gossip delegate
	dlg *delegate

	// Gossip
	memberlist *memberlist.Memberlist

	// KV API
	kvs *KVS

	// Block device for blox API
	dev *BlockDevice

	// Actual kv store
	kvstore KVStore

	grpc *grpc.Server

	hexalog *Hexalog
}

// Create creates a new fidias instance.  It inits the local node, gossip layer
// and associated delegates
func Create(conf *Config) (*Fidias, error) {
	// Coorinate client
	coord, err := vivaldi.NewClient(vivaldi.DefaultConfig())
	if err != nil {
		return nil, err
	}

	fid := &Fidias{
		conf:    conf,
		ltime:   &hexatype.LamportClock{},
		kvstore: NewInmemKVStore(),
		coord:   coord,
		grpc:    grpc.NewServer(),
	}

	// The order of initialization is important

	if err = fid.initDHT(); err != nil {
		return nil, err
	}

	if err = fid.initBlockDevice(); err != nil {
		return nil, err
	}

	if err = fid.initHexalog(); err != nil {
		return nil, err
	}

	fid.initKVS()

	fid.init()

	if err = fid.startGrpc(); err != nil {
		return nil, err
	}

	fid.memberlist, err = memberlist.Create(conf.Memberlist)

	return fid, err
}

func (fidias *Fidias) init() {

	fidias.dlg = &delegate{
		local:      fidias.local,
		coord:      fidias.coord,
		dht:        fidias.dht,
		broadcasts: make([][]byte, 0),
	}

	//
	c := fidias.conf.Memberlist
	c.Delegate = fidias.dlg
	c.Ping = fidias
	c.Events = fidias.dlg
	c.Alive = fidias.dlg
	c.Conflict = fidias.dlg
}

func (fidias *Fidias) initDHT() error {
	udpAddr, err := net.ResolveUDPAddr("udp", fidias.conf.DHT.Hostname)
	if err != nil {
		return err
	}

	ln, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}

	remote := kelips.NewUDPTransport(ln)
	fidias.dht = kelips.Create(fidias.conf.DHT, remote)
	fidias.local = fidias.dht.LocalNode()

	return nil
}

// must be called after dht is init'd
func (fidias *Fidias) initBlockDevice() error {
	ln, err := net.Listen("tcp", fidias.conf.DHT.Hostname)
	if err != nil {
		return err
	}

	dir := filepath.Join(fidias.conf.DataDir, "block")
	os.MkdirAll(dir, 0755)
	// Local
	index := device.NewInmemIndex()
	raw, err := device.NewFileRawDevice(fidias.conf.DataDir, fidias.conf.HashFunc)
	if err != nil {
		return err
	}
	dev := device.NewBlockDevice(index, raw)
	dev.SetDelegate(fidias)

	// Remote
	opts := blox.DefaultNetClientOptions(fidias.conf.HashFunc)
	remote := blox.NewNetTransport(opts)

	// Local and remote
	trans := blox.NewLocalNetTranport(fidias.conf.DHT.Hostname, remote)

	// DHT block device
	fidias.dev = NewBlockDevice(fidias.conf.Replicas, fidias.conf.HashFunc, trans)
	fidias.dev.Register(dev)
	fidias.dev.RegisterDHT(fidias.dht)

	err = trans.Start(ln.(*net.TCPListener))
	return err
}

func (fidias *Fidias) initHexalog() error {

	edir := filepath.Join(fidias.conf.DataDir, "log", "entry")
	os.MkdirAll(edir, 0755)
	entries := hexaboltdb.NewEntryStore()
	if err := entries.Open(edir); err != nil {
		return err
	}

	edir = filepath.Join(fidias.conf.DataDir, "log", "index")
	os.MkdirAll(edir, 0755)
	index := hexaboltdb.NewIndexStore()
	if err := index.Open(edir); err != nil {
		return err
	}

	//entries := hexalog.NewInMemEntryStore()
	//index := hexalog.NewInMemIndexStore()

	stable := &hexalog.InMemStableStore{}

	localTuple := kelips.TupleHost(fidias.local.Address)
	fsm := NewFSM(fidias.conf.KVPrefix, localTuple, fidias.kvstore)
	fsm.RegisterDHT(fidias.dht)

	// Network transport
	hlnet := hexalog.NewNetTransport(30*time.Second, 300*time.Second)
	hexalog.RegisterHexalogRPCServer(fidias.grpc, hlnet)

	hexlog, err := hexalog.NewHexalog(fidias.conf.Hexalog, fsm, entries, index, stable, hlnet)
	if err != nil {
		return err
	}

	trans := &localHexalogTransport{
		host:   fidias.conf.Hexalog.Hostname,
		hexlog: hexlog,
		remote: hlnet,
	}

	fidias.hexalog = &Hexalog{
		hashFunc: fidias.conf.Hexalog.Hasher,
		minVotes: fidias.conf.Hexalog.Votes,
		trans:    trans,
		jury:     &SimpleJury{dht: fidias.dht},
	}

	return nil
}

// init kvs.  hexalog needs to be init'd before this
func (fidias *Fidias) initKVS() {
	kvnet := NewNetTransport(30*time.Second, 300*time.Second)
	RegisterFidiasRPCServer(fidias.grpc, kvnet)

	kvtrans := newLocalKVTransport(fidias.conf.Hexalog.Hostname, kvnet)
	kvtrans.Register(fidias.kvstore)

	fidias.kvs = NewKVS(fidias.conf.KVPrefix, fidias.hexalog, kvtrans, fidias.dht)
}

func (fidias *Fidias) startGrpc() error {
	ln, err := net.Listen("tcp", fidias.conf.Hexalog.Hostname)
	if err != nil {
		return err
	}

	go func() {
		if er := fidias.grpc.Serve(ln); er != nil {
			log.Fatal(er)
		}
	}()

	log.Println("[INFO] Fidias started:", ln.Addr().String())
	return nil
}

// DHT returns a distributed hash table interface
func (fidias *Fidias) DHT() DHT {
	return fidias.dht
}

// KVS returns the kvs instance
func (fidias *Fidias) KVS() *KVS {
	return fidias.kvs
}

// BlockDevice returns a cluster aware block device
func (fidias *Fidias) BlockDevice() *BlockDevice {
	return fidias.dev
}

// Join joins the gossip networking using an existing node
func (fidias *Fidias) Join(existing []string) error {
	//
	_, err := fidias.memberlist.Join(existing)
	return err
}

// Shutdown performs a complete shutdown of all components
func (fidias *Fidias) Shutdown() error {
	return fmt.Errorf("TBI")
}
