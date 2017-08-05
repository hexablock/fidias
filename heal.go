package fidias

import (
	"github.com/hexablock/hexalog"
	"github.com/hexablock/log"
)

func (fidias *Fidias) heal(req *hexalog.RPCRequest) (*hexalog.FutureEntry, *ReMeta, error) {
	e := req.Entry
	opts := req.Options

	// Get the location for this node
	selfLoc, err := opts.LocationSet().GetByHost(fidias.conf.Hostname())
	if err != nil {
		return nil, nil, err
	}
	// submitter location
	// loc := opts.SourcePeer()

	// Get local key log
	keylog, err := fidias.trans.local.GetKey(e.Key)
	if err != nil {
		if keylog, err = fidias.trans.local.NewKey(e.Key, selfLoc.ID); err != nil {
			return nil, nil, err
		}
	}

	// Lookup replica locations
	locs, err := fidias.ring.LookupReplicated(e.Key, fidias.conf.Replicas)
	if err != nil {
		return nil, nil, err
	}

	tried := map[string]bool{}

	for _, loc := range locs {
		// Get the replica location chain
		_, vns, err := fidias.ring.Lookup(fidias.conf.Ring.NumSuccessors, loc.ID)
		if err != nil {
			continue
		}
		// Traverse the replica locaiton chain
		for _, vn := range vns {
			if vn.Host == fidias.conf.Hostname() {
				continue
			}

			if _, ok := tried[vn.Host]; ok {
				continue
			}
			tried[vn.Host] = true

			last := keylog.LastEntry()
			if last == nil {
				// Dont set Previous so we can signal a complete keylog download
				last = &hexalog.Entry{Key: e.Key}
			}

			_, err = fidias.trans.remote.FetchKeylog(vn.Host, last)
			if err != nil {
				log.Printf("[ERROR] key=%s vnode=%s/%x %v", e.Key, vn.Host, vn.Id, err)
				continue
			}

		}

	}

	last := keylog.LastEntry()
	if last != nil {
		log.Printf("[DEBUG] Healed curr-prev=%x.%d req-prev=%x.%d", last.Previous, last.Height, e.Previous, e.Height)
	}

	return nil, nil, nil

}
