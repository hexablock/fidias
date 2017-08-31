package fidias

import (
	"context"

	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
)

type kvitemError struct {
	loc *hexaring.Location
	kv  *hexatype.KeyValuePair
	err error
}

// GetKey gets a given key from possible locations
func (fidias *Fidias) GetKey(key []byte) (kvp *hexatype.KeyValuePair, meta *ReMeta, err error) {
	locs, err := fidias.ring.LookupReplicated(key, fidias.conf.Replicas)
	if err != nil {
		return nil, nil, err
	}
	meta = &ReMeta{PeerSet: locs}

	ll := len(locs)
	resp := make(chan *kvitemError, ll)
	ctx, cancel := context.WithCancel(context.Background())

	for _, l := range locs {

		go func(k []byte, loc *hexaring.Location) {
			kvi := &kvitemError{loc: loc}
			kvi.kv, kvi.err = fidias.trans.GetKey(ctx, loc.Host(), k)
			resp <- kvi

		}(key, l)

	}

	defer cancel()

	for i := 0; i < ll; i++ {
		kvi := <-resp
		if kvi.err == nil {
			meta.Vnode = kvi.loc.Vnode
			return
		}
	}

	err = hexatype.ErrKeyNotFound
	return
}
