package fidias

import (
	"log"

	"github.com/hexablock/blox/block"
	kelips "github.com/hexablock/go-kelips"
)

// BlockSet is the blox delegate to handle block inserts to the dht
func (fid *Fidias) BlockSet(blk block.Block) {

	tuple := kelips.TupleHost(fid.local.Address)
	if err := fid.dht.Insert(blk.ID(), tuple); err != nil {
		log.Printf("[ERROR] Failed to insert to dht: %s", err)
	}

}

// BlockRemove is the blox delegate to handle block removals from the dht
func (fid *Fidias) BlockRemove(id []byte) {
	if err := fid.dht.Delete(id); err != nil {
		log.Printf("[ERROR] Failed to delete from dht: %s", err)
	}
}
