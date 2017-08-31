package fidias

import (
	"bytes"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexalog/store"
	"github.com/hexablock/hexaring"
)

type testServer struct {
	ln   net.Listener
	g    *grpc.Server
	c    *Config
	ps   hexaring.PeerStore
	r    *hexaring.Ring
	fids *Fidias
}

func (ts *testServer) start() {
	go ts.g.Serve(ts.ln)
}

func newTestServer(addr string, peers ...string) (*testServer, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	ts := &testServer{
		ln: ln,
		g:  grpc.NewServer(),
		c:  DefaultConfig(addr),
		ps: hexaring.NewInMemPeerStore(),
	}
	ts.c.Ring.StabilizeMin = time.Duration(15 * time.Millisecond)
	ts.c.Ring.StabilizeMax = time.Duration(45 * time.Millisecond)
	ts.c.Ring.Meta = chord.Meta{"key": []byte("test")}

	for _, p := range peers {
		ts.ps.AddPeer(p)
	}

	ts.r = hexaring.New(ts.c.Ring, ts.ps, ts.g)

	idx := store.NewInMemIndexStore()
	ss := &store.InMemStableStore{}
	es := store.NewInMemEntryStore()
	ls := hexalog.NewLogStore(es, idx, ts.c.Hexalog.Hasher)
	fsm := NewInMemKeyValueFSM()

	ts.fids, err = New(ts.c, fsm, idx, es, ls, ss, ts.g)
	if err != nil {
		return nil, err
	}
	ts.start()

	if len(peers) == 0 {
		err = ts.r.Create()
	} else {
		err = ts.r.RetryJoin()
	}

	if err == nil {
		ts.fids.Register(ts.r)
	}

	return ts, err
}

func TestFidias(t *testing.T) {
	ts1, err := newTestServer("127.0.0.1:61234")
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(100 * time.Millisecond)

	ts2, err := newTestServer("127.0.0.1:61235", "127.0.0.1:61234")
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(100 * time.Millisecond)

	ts3, err := newTestServer("127.0.0.1:61236", "127.0.0.1:61235")
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(300 * time.Millisecond)

	testkey1 := []byte("testkey1")
	_, _, err = ts1.fids.NewEntry(testkey1)
	if err != nil {
		t.Fatal(err)
	}

	entry, opt, err := ts2.fids.NewEntry(testkey1)
	if err != nil {
		t.Fatal(err)
	}
	//&hexatype.RequestOptions{PeerSet: remeta.PeerSet}
	ballot, err := ts3.fids.ProposeEntry(entry, opt)
	if err != nil {
		t.Fatal(err)
	}

	if err = ballot.Wait(); err != nil {
		t.Fatal(err)
	}

	fe := ballot.Future()
	if _, err = fe.Wait(1 * time.Second); err != nil {
		t.Fatal(err)
	}

	id := entry.Hash(ts2.c.Hasher().New())
	if _, _, err = ts2.fids.GetEntry(entry.Key, id); err != nil {
		t.Fatal(err)
	}

	ki, err := ts3.fids.trans.local.GetKey(entry.Key)
	if err != nil {
		t.Fatal(err)
	}

	kidx := ki.GetIndex()
	if bytes.Compare(kidx.Last(), id) != 0 {
		t.Fatal("id mismatch")
	}

	// New node joining
	<-time.After(10 * time.Millisecond)
	ts4, err := newTestServer("127.0.0.1:61237", "127.0.0.1:61234")
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(200 * time.Millisecond)

	ts1.fids.shutdownWait()
	ts2.fids.shutdownWait()
	ts3.fids.shutdownWait()
	ts4.fids.shutdownWait()
}
