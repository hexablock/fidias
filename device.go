package fidias

import (
	"errors"
	"fmt"
	"hash"

	"github.com/hexablock/blox"
	"github.com/hexablock/blox/block"
	"github.com/hexablock/blox/device"
)

var errBloxAddrMissing = errors.New("blox address missing")

// BlockDevice implements the blox.BlockDevice interface backed by a dht to
// distribute blocks into the cluster.  The filesystem uses this as its
// underlying device.
type BlockDevice struct {
	dht DHT

	// Min number of block replicas
	replicas int

	// hash function to use
	hashFunc func() hash.Hash

	// Blox transport.  This can be either LocalNetTransport for cluster members
	// or a blox.NetClient one for clients
	trans blox.Transport
}

// NewBlockDevice inits a new Device that implements a BlockDevice that is
// leverages the dht with the given replica count, hash function and blox
// transport.
func NewBlockDevice(replicas int, hashFunc func() hash.Hash, trans blox.Transport) *BlockDevice {
	return &BlockDevice{
		replicas: replicas,
		hashFunc: hashFunc,
		trans:    trans,
	}
}

// Register registers the block device to the transport.  This is used in the
// case where the node is a member of the cluster rather than just a client
func (dev *BlockDevice) Register(blkDev *device.BlockDevice) {
	dev.trans.Register(blkDev)
}

// RegisterDHT registers the DHT to the device.  This device is only usable once
// a call to register has been made.
func (dev *BlockDevice) RegisterDHT(dht DHT) {
	dev.dht = dht
}

// Stats is to satisfy the interface
func (dev *BlockDevice) Stats() *device.Stats {
	return nil
}

// Hasher returns the hash function generator for hash ids for the device
func (dev *BlockDevice) Hasher() func() hash.Hash {
	return dev.hashFunc
}

// BlockExists returns true if the block exists on any one of the assigned nodes
func (dev *BlockDevice) BlockExists(id []byte) (bool, error) {
	nodes, err := dev.dht.LookupGroupNodes(id)
	if err != nil {
		return false, err
	}
	for _, node := range nodes {
		if ok, err := dev.trans.BlockExists(node.Host(), id); err == nil && ok {
			return true, nil
		}
	}
	return false, nil
}

// SetBlock writes the block to the device
func (dev *BlockDevice) SetBlock(blk block.Block) ([]byte, error) {
	nodes, err := dev.dht.LookupGroupNodes(blk.ID())
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no peers found")
	}

	var id []byte
	for _, loc := range nodes {

		bid, er := dev.trans.SetBlock(loc.Host(), blk)
		if er != nil {
			err = er
		} else {
			// Latest set block id
			id = bid
		}

	}

	//log.Printf("[DEBUG] Device.SetBlock id=%x type=%s replicas=%d error='%v'",
	//	blk.ID(), blk.Type(), len(nodes), err)

	return id, err
}

// GetBlock gets a block from the device
func (dev *BlockDevice) GetBlock(id []byte) (block.Block, error) {
	locs, err := dev.dht.Lookup(id)
	if err != nil {
		return nil, err
	}

	var blk block.Block
	for _, loc := range locs {

		if blk, err = dev.trans.GetBlock(loc.Host(), id); err == nil {
			return blk, nil
		}

	}

	return nil, err
}

// RemoveBlock submits a request to remove a block on the device and all replicas
func (dev *BlockDevice) RemoveBlock(id []byte) error {
	locs, err := dev.dht.Lookup(id)
	if err != nil {
		return err
	}

	for _, loc := range locs {

		if er := dev.trans.RemoveBlock(loc.Host(), id); er != nil {
			err = er
		}

	}

	return err
}

// Close shutdowns the underlying network transport
func (dev *BlockDevice) Close() error {
	return dev.trans.Shutdown()
}
