package fidias

import (
	"fmt"
	"io"
	"time"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
)

// Hexalog is a ring/cluster aware Hexalog.
type Hexalog struct {
	// hexalog config
	conf *hexalog.Config

	// proposal retry interval
	retryInt time.Duration

	locator *hexaring.Ring

	hexlog *hexalog.Hexalog
	trans  *localHexalogTransport
}

// remote must be registered to grpc before init'ing hexalog
func NewHexalog(conf *hexalog.Config, logstore *hexalog.LogStore, stable hexalog.StableStore, afsm KeyValueFSM, remote *hexalog.NetTransport) (*Hexalog, error) {
	// Init FSM
	var fsm KeyValueFSM
	if afsm == nil {
		fsm = &DummyFSM{}
	} else {
		fsm = afsm
	}
	// Make it available for use
	if err := fsm.Open(); err != nil {
		return nil, err
	}

	// maxIdle := 3 * time.Minute
	// reapInterval := 30 * time.Second
	retryInt := 10 * time.Millisecond
	// remote := hexalog.NewNetTransport(reapInterval, maxIdle)

	hexlog, err := hexalog.NewHexalog(conf, fsm, logstore, stable, remote)
	if err != nil {
		return nil, err
	}
	remote.Register(hexlog)

	trans := &localHexalogTransport{
		host:     conf.Hostname,
		logstore: logstore,
		remote:   remote,
	}

	hexl := &Hexalog{
		conf:     conf,
		hexlog:   hexlog,
		retryInt: retryInt,
		trans:    trans,
	}

	return hexl, nil
}

// Register registers the locator to hexalog
func (hexlog *Hexalog) Register(locator *hexaring.Ring) {
	hexlog.locator = locator
}

// MinVotes returns the minimum number of required votes for a proposal and commit
func (hexlog *Hexalog) MinVotes() int {
	return hexlog.conf.Votes
}

// NewEntry returns a new Entry for the given key from Hexalog.  It returns an error if
// the node is not part of the location set or a lookup error occurs
func (hexlog *Hexalog) NewEntry(key []byte) (*hexatype.Entry, *hexatype.RequestOptions, error) {
	// Lookup locations for this key
	locs, err := hexlog.locator.LookupReplicated(key, hexlog.MinVotes())
	if err != nil {
		return nil, nil, err
	}

	// Check and set source index
	opt := &hexatype.RequestOptions{SourceIndex: -1, PeerSet: locs}
	for i, v := range locs {
		if v.Host() == hexlog.conf.Hostname {
			opt.SourceIndex = int32(i)
			break
		}
	}
	// Check we are a member of the set
	if opt.SourceIndex < 0 {
		return nil, opt, fmt.Errorf("host not in set: %s", hexlog.conf.Hostname)
	}

	return hexlog.hexlog.New(key), opt, nil
}

// ProposeEntry finds locations for the entry and proposes it to those locations.  It retries
// the specified number of times before returning.  It returns a ballot that can be waited on
// for the entry to be applied or an error
func (hexlog *Hexalog) ProposeEntry(entry *hexatype.Entry, opts *hexatype.RequestOptions) (ballot *hexalog.Ballot, err error) {
	retries := int(opts.Retries)
	if retries < 1 {
		retries = 1
	}

	for i := 0; i < retries; i++ {
		// Propose with retries.  Retry only on a ErrPreviousHash error
		if ballot, err = hexlog.hexlog.Propose(entry, opts); err == nil {
			return
		} else if err == hexatype.ErrPreviousHash {
			time.Sleep(hexlog.retryInt)
		} else {
			return
		}

	}

	return
}

// GetEntry tries to get an entry from the ring.  It gets the replica locations and queries
// upto the max allowed successors for each location.
func (hexlog *Hexalog) GetEntry(key, id []byte) (entry *hexatype.Entry, meta *ReMeta, err error) {
	meta = &ReMeta{}
	_, err = hexlog.locator.ScourReplicatedKey(key, hexlog.MinVotes(), func(vn *chord.Vnode) error {
		ent, er := hexlog.trans.GetEntry(vn.Host, key, id)
		if er == nil {
			entry = ent
			meta.Vnode = vn
			return io.EOF
		}

		return nil
	})

	// We found the entry.
	if err == io.EOF {
		err = nil
	} else if entry == nil {
		err = hexatype.ErrEntryNotFound
	}

	return
}

// Leader returns the leader of the given location set from the underlying log.
func (hexlog *Hexalog) Leader(key []byte, locs hexaring.LocationSet) (*hexalog.KeyLeader, error) {
	return hexlog.hexlog.Leader(key, locs)
}

func (hexlog *Hexalog) Heal(key []byte, opts *hexatype.RequestOptions) error {
	return hexlog.hexlog.Heal(key, opts)
}
