package fidias

import (
	"fmt"
	"time"

	"github.com/hexablock/blox"
	kelips "github.com/hexablock/go-kelips"
	"github.com/hexablock/hexalog"
)

// Client is a fidias client.  It is used by non-partiicating client users
type Client struct {
	conf *Config

	// Client coordinates
	//coord *vivaldi.Client

	// Lookup, Insert, Delete
	dht DHT

	// New, Propose
	wal WAL

	// DHT aware block device - Get, Set, Remove
	dev *BlockDevice

	kvs *KVS
}

func NewClient(conf *Config) (client *Client, err error) {
	client = &Client{conf: conf}
	if len(conf.Peers) == 0 {
		err = fmt.Errorf("no peers provided")
		return
	}

	// coord, err := vivaldi.NewClient(vivaldi.DefaultConfig())
	// if err != nil {
	// 	return nil, err
	// }
	// client.coord = coord

	if err = client.initDHT(); err != nil {
		return
	}

	client.initBlockDevice()
	client.initWAL()
	client.initKVS()

	return client, nil
}

func (client *Client) initDHT() error {

	dht, err := kelips.NewClient(client.conf.Peers...)
	if err == nil {
		client.dht = dht
	}

	return err
}

func (client *Client) initBlockDevice() {
	opt := blox.DefaultNetClientOptions(client.conf.HashFunc)
	trans := blox.NewNetTransport(opt)

	client.dev = NewBlockDevice(client.conf.Replicas, client.conf.HashFunc, trans)
	client.dev.RegisterDHT(client.dht)
}

func (client *Client) initWAL() {
	jury := &SimpleJury{dht: client.dht}

	trans := hexalog.NewNetTransport(30*time.Second, 180*time.Second)

	client.wal = &Hexalog{
		jury:     jury,
		minVotes: client.conf.Hexalog.Votes,
		hashFunc: client.conf.HashFunc,
		trans:    trans,
	}
}

func (client *Client) initKVS() {
	trans := NewNetTransport(30*time.Second, 300*time.Second)
	client.kvs = NewKVS(client.conf.KVPrefix, client.wal, trans, client.dht)
}

func (client *Client) Blox() *blox.Blox {
	return blox.NewBlox(client.dev)
}

func (client *Client) KVS() *KVS {
	return client.kvs
}
