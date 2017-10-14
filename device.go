package fidias

import (
	"errors"

	"github.com/hexablock/blox"
	"github.com/hexablock/blox/block"
	"github.com/hexablock/blox/device"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

var errBloxAddrMissing = errors.New("blox address missing")

// RingDevice implements the blox.BlockDevice interface backed by hexaring to distribute
// blocks into the cluster.  The filesystem uses this as its underlying device.
type RingDevice struct {
	dht DHT
	// Number of block replicas
	replicas int
	hasher   hexatype.Hasher
	dev      *device.BlockDevice
	// Blox transport with fast routing
	trans blox.Transport
}

// NewRingDevice inits a new RingDevice that implements a BlockDevice with the given
// replica count, hash function and blox transport.
func NewRingDevice(replicas int, hasher hexatype.Hasher, dev *device.BlockDevice, trans blox.Transport) *RingDevice {
	return &RingDevice{
		replicas: replicas,
		hasher:   hasher,
		dev:      dev,
		trans:    trans,
	}
}

// Stats returns device statistics
func (dev *RingDevice) Stats() *device.Stats {
	return dev.dev.Stats()
}

// Hasher returns the hash function generator for hash ids for the device
func (dev *RingDevice) Hasher() hexatype.Hasher {
	return dev.hasher
}

// SetBlock writes the block to the device
func (dev *RingDevice) SetBlock(blk block.Block) ([]byte, error) {

	locs, err := dev.dht.LookupReplicatedHash(blk.ID(), dev.replicas)
	//log.Printf("Setting block id=%x %v", blk.ID(), err)
	if err != nil {
		return nil, err
	}

	//log.Printf("Setting block id=%x", blk.ID())

	var id []byte
	//loc := locs[0]
	for _, loc := range locs {
		meta := loc.Vnode.Metadata()
		host, ok := meta["blox"]
		if !ok {
			return nil, errBloxAddrMissing
		}

		//log.Printf("RingDevice.SetBlock id=%x", blk.ID())

		i, er := dev.trans.SetBlock(string(host), blk)
		if er != nil {
			//log.Println("[ERROR] RingDevice.SetBlock", er)
			err = er
		} else {
			id = i
		}

		// TODO:
		//break
	}

	log.Printf("[DEBUG] RingDevice.SetBlock id=%x type=%s error='%v'", blk.ID(), blk.Type(), err)

	return id, err
}

// GetBlock gets a block from the device
func (dev *RingDevice) GetBlock(id []byte) (block.Block, error) {
	locs, err := dev.dht.LookupReplicatedHash(id, dev.replicas)
	if err != nil {
		return nil, err
	}

	var blk block.Block
	for _, loc := range locs {
		meta := loc.Vnode.Metadata()
		host, ok := meta["blox"]
		if !ok {
			return nil, errBloxAddrMissing
		}

		if blk, err = dev.trans.GetBlock(string(host), id); err == nil {
			return blk, nil
		}
	}

	return nil, err
}

// RemoveBlock submits a request to remove a block on the device and all replicas
func (dev *RingDevice) RemoveBlock(id []byte) error {
	locs, err := dev.dht.LookupReplicatedHash(id, dev.replicas)
	if err != nil {
		return err
	}

	for _, loc := range locs {
		meta := loc.Vnode.Metadata()
		host, ok := meta["blox"]
		if !ok {
			return errBloxAddrMissing
		}

		if er := dev.trans.RemoveBlock(string(host), id); er != nil {
			err = er
		}
		// TODO:
		//break
	}

	return err
}

// Close shutdowns the underlying network transport
func (dev *RingDevice) Close() error {
	return dev.trans.Shutdown()
}

// RegisterDHT registers the DHT ring device.  This device is only usable once a call
// to register has been made.
func (dev *RingDevice) RegisterDHT(dht DHT) {
	dev.dht = dht
}
