package fidias

import (
	"context"
	"hash"
	"time"

	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

// Jury implements an interface to get participants for an entry proposal. These
// are the peers participating in the voting process
type Jury interface {
	Participants(key []byte, min int) ([]*hexalog.Participant, error)
}

// Hexalog is a network aware Hexalog.  It implements selecting the
// participants from the network for consistency
type Hexalog struct {
	// hash function to use
	hashFunc func() hash.Hash

	// min votes for any log entry
	minVotes int

	// Hexalog
	trans WALTransport

	// Jury selector for voting rounds
	jury Jury
}

// NewHexalog inits a new fidias hexalog instance attached to the ring.  Remote must
// be registered to grpc before init'ing hexalog
func NewHexalog(minVotes int, hf func() hash.Hash, trans WALTransport) (*Hexalog, error) {

	hxl := &Hexalog{
		minVotes: minVotes,
		hashFunc: hf,
		trans:    trans,
	}

	return hxl, nil
}

// NewEntry returns a new Entry for the given key from Hexalog.  It returns an
// error if the node is not part of the location set or a lookup error occurs
func (hexlog *Hexalog) NewEntry(key []byte) (*hexalog.Entry, []*hexalog.Participant, error) {
	peers, err := hexlog.jury.Participants(key, hexlog.minVotes)
	if err != nil {
		return nil, nil, err
	}

	opt := &hexalog.RequestOptions{}
	var entry *hexalog.Entry

	for _, loc := range peers {
		if entry, err = hexlog.trans.NewEntry(loc.Host, key, opt); err == nil {
			return entry, peers, nil
		}
	}

	return nil, peers, err
}

// NewEntryFrom creates a new entry based on the given entry.  It uses the
// given height and previous hash of the entry to determine the values for
// the new entry.  This is essentially a compare and set
func (hexlog *Hexalog) NewEntryFrom(entry *hexalog.Entry) (*hexalog.Entry, []*hexalog.Participant, error) {
	peers, err := hexlog.jury.Participants(entry.Key, hexlog.minVotes)
	if err != nil {
		return nil, nil, err
	}

	nentry := &hexalog.Entry{
		Key:       entry.Key,
		Previous:  entry.Hash(hexlog.hashFunc()),
		Height:    entry.Height + 1,
		Timestamp: uint64(time.Now().UnixNano()),
	}

	return nentry, peers, nil
}

// GetEntry tries to get an entry from the network from all known locations
func (hexlog *Hexalog) GetEntry(key, id []byte) (*hexalog.Entry, error) {
	peers, err := hexlog.jury.Participants(key, hexlog.minVotes)
	if err != nil {
		return nil, err
	}

	opt := &hexalog.RequestOptions{}

	for _, p := range peers {
		ent, er := hexlog.trans.GetEntry(p.Host, key, id, opt)
		if er == nil {
			return ent, nil
		}
		err = er
	}

	return nil, err
}

// ProposeEntry finds locations for the entry and proposes it to those locations
// It retries the specified number of times before returning.  It returns a an
// entry id on success and error otherwise
func (hexlog *Hexalog) ProposeEntry(entry *hexalog.Entry, opts *hexalog.RequestOptions, retries int, retryInt time.Duration) (eid []byte, stats *WriteStats, err error) {
	if retries < 1 {
		retries = 1
	}

	if retryInt == 0 {
		retryInt = 30 * time.Millisecond
	}

	log.Printf("[DEBUG] Proposing key=%s participants=%d", entry.Key, len(opts.PeerSet))

	for i := 0; i < retries; i++ {
		// Propose with retries.  Retry only on a ErrPreviousHash error
		var resp *hexalog.ReqResp
		if resp, err = hexlog.trans.ProposeEntry(context.Background(), opts.PeerSet[0].Host, entry, opts); err == nil {

			eid = entry.Hash(hexlog.hashFunc())
			stats = &WriteStats{
				BallotTime:   time.Duration(resp.BallotTime),
				ApplyTime:    time.Duration(resp.ApplyTime),
				Participants: opts.PeerSet,
			}
			return

		} else if err == hexatype.ErrPreviousHash {
			time.Sleep(retryInt)
		} else {
			return
		}

	}

	return
}
