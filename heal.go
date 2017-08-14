package fidias

import (
	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

//
// TODO: The algorithm needs improvement to perform actual reconciliation by checking all
// members of the peer set and reconciling as needed
//

// req.ID may or may not be available depending on where the heal was initiated
func (fidias *Fidias) heal(req *hexatype.ReqResp) (*hexalog.FutureEntry, *ReMeta, error) {
	e := req.Entry
	opts := req.Options
	host := fidias.conf.Hostname()

	// Get the location for this node
	selfLoc, err := opts.LocationSet().GetByHost(host)
	if err != nil {
		return nil, nil, err
	}

	// Get local key log
	keylog, err := fidias.trans.local.GetKey(e.Key)
	if err != nil {
		if keylog, err = fidias.trans.local.NewKey(e.Key, selfLoc.ID); err != nil {
			return nil, nil, err
		}
	}

	_, err = fidias.ring.Scour(opts.PeerSet, func(vn *chord.Vnode) error {
		// Skip self
		if vn.Host == host {
			return nil
		}

		// Get current last entry
		last := keylog.LastEntry()
		if last == nil {
			// Dont set Previous so we can signal a complete keylog download
			last = &hexatype.Entry{Key: e.Key}
		}

		if _, er := fidias.trans.remote.FetchKeylog(vn.Host, last); er != nil {
			log.Printf("[ERROR] Failed to fetch KeyLog key=%s vnode=%s/%x %v", e.Key, vn.Host, vn.Id, er)
		}

		return nil
	})

	// last := keylog.LastEntry()
	// if last != nil {
	// 	log.Printf("[DEBUG] Healed key=%s curr-prev=%x.%d req-prev=%x.%d", last.Key, last.Previous, last.Height, e.Previous, e.Height)
	// }

	return nil, nil, err

}

func (fidias *Fidias) startHealer() {
	// Get the read-only heal channel from the log
	heal := fidias.hexlog.Heal()
	for req := range heal {
		if _, _, err := fidias.heal(req); err != nil {
			log.Printf("[ERROR] Failed to heal key=%s height=%d id=%x error='%v'",
				req.Entry.Key, req.Entry.Height, req.ID, err)
		}
	}

	fidias.shutdown <- struct{}{}
}
