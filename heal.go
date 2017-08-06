package fidias

import (
	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/log"
)

func (fidias *Fidias) heal(req *hexalog.ReqResp) (*hexalog.FutureEntry, *ReMeta, error) {
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

	_, err = fidias.ring.Orbit(e.Key, fidias.conf.Replicas, func(vn *chord.Vnode) error {
		// Skip self
		if vn.Host == fidias.conf.Hostname() {
			return nil
		}

		last := keylog.LastEntry()
		if last == nil {
			// Dont set Previous so we can signal a complete keylog download
			last = &hexalog.Entry{Key: e.Key}
		}

		if _, er := fidias.trans.remote.FetchKeylog(vn.Host, last); er != nil {
			log.Printf("[ERROR] key=%s vnode=%s/%x %v", e.Key, vn.Host, vn.Id, er)
		}

		return nil
	})

	last := keylog.LastEntry()
	if last != nil {
		log.Printf("[DEBUG] Healed key=%s curr-prev=%x.%d req-prev=%x.%d", last.Key, last.Previous, last.Height, e.Previous, e.Height)
	}

	return nil, nil, err

}

func (fidias *Fidias) startHealer() {
	// Get the heal channel from the log
	healCh := fidias.hexlog.Heal()

	for req := range healCh {
		if _, _, err := fidias.heal(req); err != nil {
			log.Printf("[ERROR] Failed to heal key=%s height=%d id=%x error='%v'",
				req.Entry.Key, req.Entry.Height, req.ID, err)
		}
	}

	fidias.shutdown <- struct{}{}
}
