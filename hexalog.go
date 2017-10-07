package fidias

import (
	"io"
	"time"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
)

// Hexalog is a ring/cluster aware Hexalog.
type Hexalog struct {
	conf     *hexalog.Config        // Hexalog config
	retryInt time.Duration          // Proposal retry interval
	hexlog   *hexalog.Hexalog       // Hexalog instance
	trans    *localHexalogTransport // Hexalog local and remote transport

	dht DHT
}

// NewHexalog inits a new fidias hexalog instance attached to the ring.  Remote must
// be registered to grpc before init'ing hexalog
func NewHexalog(conf *Config, logstore *hexalog.LogStore, stable hexalog.StableStore, f *FSM, remote *hexalog.NetTransport) (*Hexalog, error) {
	// Init FSM
	var fsm *FSM
	if f == nil {
		fsm = NewFSM(conf.KeyValueNamespace, conf.FileSystemNamespace)
	} else {
		fsm = f
	}
	// Make it available for use
	if err := fsm.Open(); err != nil {
		return nil, err
	}

	retryInt := 10 * time.Millisecond
	// remote := hexalog.NewNetTransport(reapInterval, maxIdle)

	hexlog, err := hexalog.NewHexalog(conf.Hexalog, fsm, logstore, stable, remote)
	if err != nil {
		return nil, err
	}
	remote.Register(hexlog)

	trans := &localHexalogTransport{
		host:   conf.Hostname(),
		hexlog: hexlog,
		store:  logstore,
		remote: remote,
	}

	hexl := &Hexalog{
		conf:     conf.Hexalog,
		hexlog:   hexlog,
		retryInt: retryInt,
		trans:    trans,
	}

	return hexl, nil
}

// RegisterDHT registers the DHT to hexalog
func (hexlog *Hexalog) RegisterDHT(dht DHT) {
	hexlog.dht = dht
}

// MinVotes returns the minimum number of required votes for a proposal and commit
func (hexlog *Hexalog) MinVotes() int {
	return hexlog.conf.Votes
}

// NewEntry returns a new Entry for the given key from Hexalog.  It returns an error if
// the node is not part of the location set or a lookup error occurs
func (hexlog *Hexalog) NewEntry(key []byte) (*hexatype.Entry, *hexatype.RequestOptions, error) {
	// Lookup locations for this key
	locs, err := hexlog.dht.LookupReplicated(key, hexlog.MinVotes())
	if err != nil {
		return nil, nil, err
	}
	opt := hexatype.DefaultRequestOptions()
	opt.PeerSet = locs
	// if _, err := locs.GetByHost(hexlog.conf.Hostname); err != nil {
	// 	return nil, opt, err
	// }

	var entry *hexatype.Entry
	for i, loc := range locs {
		if entry, err = hexlog.trans.NewEntry(loc.Host(), key); err == nil {
			opt.SourceIndex = int32(i)
			return entry, opt, nil
		}
	}

	return nil, opt, err
}

// NewEntryFrom creates a new entry based on the given entry.  It uses the
// given height and previous hash of the entry to determine the values for
// the new entry.  This is essentially a compare and set
func (hexlog *Hexalog) NewEntryFrom(entry *hexatype.Entry) (*hexatype.Entry, *hexatype.RequestOptions, error) {

	// Lookup locations for this key
	locs, err := hexlog.dht.LookupReplicated(entry.Key, hexlog.MinVotes())
	if err != nil {
		return nil, nil, err
	}
	opt := hexatype.DefaultRequestOptions()
	opt.PeerSet = locs
	//opt := &hexatype.RequestOptions{SourceIndex: 0, PeerSet: locs}
	// if _, err := locs.GetByHost(hexlog.conf.Hostname); err != nil {
	// 	return nil, opt, err
	// }

	nentry := &hexatype.Entry{
		Key:       entry.Key,
		Previous:  entry.Hash(hexlog.conf.Hasher.New()),
		Height:    entry.Height + 1,
		Timestamp: uint64(time.Now().UnixNano()),
	}

	return nentry, opt, nil
}

// ProposeEntry finds locations for the entry and proposes it to those locations.  It retries
// the specified number of times before returning.  It returns a ballot that can be waited on
// for the entry to be applied or an error
func (hexlog *Hexalog) ProposeEntry(entry *hexatype.Entry, opts *hexatype.RequestOptions) (err error) {

	retries := int(opts.Retries)
	if retries < 1 {
		retries = 1
	}

	for i := 0; i < retries; i++ {
		// Propose with retries.  Retry only on a ErrPreviousHash error
		//if ballot, err = hexlog.hexlog.Propose(entry, opts); err == nil {
		if err = hexlog.trans.ProposeEntry(opts.PeerSet[0].Host(), entry, opts); err == nil {
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
	_, err = hexlog.dht.ScourReplicatedKey(key, hexlog.MinVotes(), func(vn *chord.Vnode) error {
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

// Heal submits a heal request for the given key to the local note.  It consults the supplied
// PeerSet in order to perform the heal.
func (hexlog *Hexalog) Heal(key []byte, opts *hexatype.RequestOptions) error {
	return hexlog.hexlog.Heal(key, opts)
}

func (hexlog *Hexalog) Stats() *hexalog.Stats {
	return hexlog.hexlog.Stats()
}
