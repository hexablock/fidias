package fidias

import (
	"errors"

	"github.com/hexablock/blox"
	"github.com/hexablock/blox/block"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

// Locator implements an interface to perform lookups against the cluster
// type Locator interface {
// 	LookupReplicatedHash(id []byte, n int) (hexaring.LocationSet, error)
// 	LookupReplicated(key []byte, n int) (hexaring.LocationSet, error)
// 	ScourReplicatedKey(key []byte, r int, cb func(*chord.Vnode) error) (int, error)
// }
//
var errBloxAddrMissing = errors.New("blox address missing")

// RingDevice implements the blox.BlockDevice interface backed by hexaring to distribute
// blocks into the cluster.  The filesystem uses this as its underlying device.
type RingDevice struct {
	locator *hexaring.Ring
	// Number of block replicas
	replicas int
	hasher   hexatype.Hasher
	// Blox transport with fast routing
	trans blox.Transport
}

// NewRingDevice inits a new RingDevice that implements a BlockDevice with the given
// replica count, hash function and blox transport.
func NewRingDevice(replicas int, hasher hexatype.Hasher, trans blox.Transport) *RingDevice {
	return &RingDevice{
		replicas: replicas,
		hasher:   hasher,
		trans:    trans,
	}
}

// Hasher returns the hash function generator for hash ids for the device
func (dev *RingDevice) Hasher() hexatype.Hasher {
	return dev.hasher
}

// SetBlock writes the block to the device
func (dev *RingDevice) SetBlock(blk block.Block) ([]byte, error) {

	locs, err := dev.locator.LookupReplicatedHash(blk.ID(), dev.replicas)
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
	locs, err := dev.locator.LookupReplicatedHash(id, dev.replicas)
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
	locs, err := dev.locator.LookupReplicatedHash(id, dev.replicas)
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

// Register registers the hexaing to the ring device.  This device is only usable once a call
// to register has been made.
func (dev *RingDevice) Register(locator *hexaring.Ring) {
	dev.locator = locator
}
