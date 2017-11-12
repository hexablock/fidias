package fidias

import (
	"fmt"
	"time"

	kelips "github.com/hexablock/go-kelips"
	"github.com/hexablock/phi"
)

// Fidias is core engine for a cluster member/participant it runs a server and
// client components
type Fidias struct {
	conf *Config

	kvstore KVStore

	fsm phi.FSM

	phi *phi.Phi

	kvs *KVS
}

// Create creates a new fidias instance.  It inits the local node, gossip layer
// and associated delegates
func Create(conf *Config) (*Fidias, error) {

	fid := &Fidias{
		conf:    conf,
		kvstore: NewInmemKVStore(),
	}

	localTuple := kelips.NewTupleHost(conf.Phi.DHT.AdvertiseHost)
	fid.fsm = NewFSM(conf.KVPrefix, localTuple, fid.kvstore)

	kvnet := NewNetTransport(30*time.Second, 300*time.Second)
	RegisterFidiasRPCServer(fid.conf.Phi.GRPCServer, kvnet)

	kvtrans := newLocalKVTransport(fid.conf.Phi.Hexalog.AdvertiseHost, kvnet)
	kvtrans.Register(fid.kvstore)

	ph, err := phi.Create(conf.Phi, fid.fsm)
	if err != nil {
		return nil, err
	}

	fid.phi = ph
	fid.kvs = NewKVS(fid.conf.KVPrefix, fid.phi.WAL(), kvtrans, fid.phi.DHT())
	kvnet.kvs = fid.kvs

	return fid, nil
}

func (fidias *Fidias) Join(existing []string) error {
	return fidias.phi.Join(existing)
}

// DHT returns a distributed hash table interface
func (fidias *Fidias) DHT() phi.DHT {
	return fidias.phi.DHT()
}

// BlockDevice returns a cluster aware block device
func (fidias *Fidias) BlockDevice() *phi.BlockDevice {
	return fidias.phi.BlockDevice()
}

// WAL returns the write-ahead-log for consistent operations
func (fidias *Fidias) WAL() phi.WAL {
	return fidias.phi.WAL()
}

// KVS returns the kvs instance
func (fidias *Fidias) KVS() *KVS {
	return fidias.kvs
}

func (fidias *Fidias) Shutdown() error {
	return fmt.Errorf("TBI")
}
