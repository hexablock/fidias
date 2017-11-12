package fidias

import (
	"context"
	"fmt"
	"time"

	"github.com/hexablock/blox"
	kelips "github.com/hexablock/go-kelips"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/phi"
)

type KV struct {
	walHost string

	kvs  *KVS
	pool *outPool
}

func (kv *KV) Set(kvp *KVPair, wo *WriteOptions) (*KVPair, *WriteStats, error) {
	conn, err := kv.pool.getConn(kv.walHost)
	if err != nil {
		return nil, nil, err
	}
	defer kv.pool.returnConn(conn)

	resp, err := conn.client.SetRPC(context.Background(), &WriteRequest{KV: kvp, Options: wo})
	if err != nil {
		return nil, nil, err
	}

	return resp.KV, resp.Stats, nil
}

func (kv *KV) CASet(kvp *KVPair, mod []byte, wo *WriteOptions) (*KVPair, *WriteStats, error) {
	conn, err := kv.pool.getConn(kv.walHost)
	if err != nil {
		return nil, nil, err
	}
	defer kv.pool.returnConn(conn)

	req := &WriteRequest{KV: kvp, Options: wo}
	req.KV.Modification = mod
	resp, err := conn.client.CASetRPC(context.Background(), req)
	if err != nil {
		return nil, nil, err
	}

	return resp.KV, resp.Stats, nil
}

func (kv *KV) Remove(key []byte, wo *WriteOptions) (*WriteStats, error) {
	conn, err := kv.pool.getConn(kv.walHost)
	if err != nil {
		return nil, err
	}
	defer kv.pool.returnConn(conn)

	req := &WriteRequest{KV: &KVPair{Key: key}, Options: wo}
	resp, err := conn.client.RemoveRPC(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp.Stats, nil
}

func (kv *KV) CARemove(key []byte, mod []byte, wo *WriteOptions) (*WriteStats, error) {
	conn, err := kv.pool.getConn(kv.walHost)
	if err != nil {
		return nil, err
	}
	defer kv.pool.returnConn(conn)

	req := &WriteRequest{KV: &KVPair{Key: key, Modification: mod}, Options: wo}
	resp, err := conn.client.CARemoveRPC(context.Background(), req)
	if err != nil {
		return nil, err
	}

	return resp.Stats, err
}

func (kv *KV) Get(key []byte, opt *ReadOptions) (*KVPair, *ReadStats, error) {
	return kv.kvs.Get(key, opt)
}

// List retrieves dir files from all the hosts owning it
func (kv *KV) List(dir []byte, opt *ReadOptions) ([]*KVPair, *ReadStats, error) {
	return kv.kvs.List(dir, opt)
}

// Client is a fidias client.  It is used by non-partiicating client users
type Client struct {
	conf *Config

	// Lookup, Insert, Delete
	dht phi.DHT

	// New, Propose
	wal phi.WAL

	// DHT aware block device - Get, Set, Remove
	dev *phi.BlockDevice

	kvs *KVS

	pool *outPool
}

func NewClient(conf *Config) (client *Client, err error) {
	client = &Client{conf: conf, pool: newOutPool(300*time.Second, 45*time.Second)}

	if len(conf.Peers) == 0 {
		err = fmt.Errorf("peers not provided")
		return
	}

	c := conf.Phi.Hexalog
	if c.AdvertiseHost == "" {
		err = fmt.Errorf("wal host not provided")
		return
	}

	if err = client.initDHT(); err != nil {
		return
	}

	client.initBlockDevice()

	// Init wal
	ltrans := hexalog.NewNetTransport(30*time.Second, 300*time.Second)
	client.wal = phi.NewHexalog(ltrans, c.Votes, c.Hasher)

	// transport to for individual host request to get keys and list dirs
	kvtrans := NewNetTransport(30*time.Second, 300*time.Second)
	client.kvs = NewKVS(conf.KVPrefix, client.wal, kvtrans, client.dht)

	return client, nil
}

func (client *Client) Blox() *blox.Blox {
	return blox.NewBlox(client.dev)
}

// KV returns a key-value client interface
func (client *Client) KV() *KV {
	kv := &KV{
		walHost: client.conf.Phi.Hexalog.AdvertiseHost,
		kvs:     client.kvs,
		pool:    client.pool,
	}
	return kv
}

func (client *Client) initDHT() error {

	dht, err := kelips.NewClient(client.conf.Peers...)
	if err == nil {
		client.dht = dht
	}

	return err
}

func (client *Client) initBlockDevice() {
	c := client.conf

	opt := blox.DefaultNetClientOptions(c.Phi.HashFunc)
	trans := blox.NewNetTransport(opt)

	client.dev = phi.NewBlockDevice(c.Phi.Replicas, c.Phi.HashFunc, trans)
	client.dev.RegisterDHT(client.dht)
}
