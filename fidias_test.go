package fidias

import (
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexalog/store"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
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

	ts.fids, err = New(ts.c, nil, idx, es, ls, ss, ts.g)
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

	ts3, err := newTestServer("127.0.0.1:61236", "127.0.0.1:61234")
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(100 * time.Millisecond)

	testkey1 := []byte("testkey1")

	_, _, err = ts1.fids.NewEntry(testkey1)
	if err != nil {
		t.Fatal(err)
	}

	entry, remeta, err := ts2.fids.NewEntry(testkey1)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = ts3.fids.ProposeEntry(entry, &hexatype.RequestOptions{PeerSet: remeta.PeerSet}); err != nil {
		t.Fatal(err)
	}

	<-time.After(10 * time.Millisecond)
	ts4, err := newTestServer("127.0.0.1:61237", "127.0.0.1:61234")
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(200 * time.Millisecond)

	// locs, err := ts4.fids.ring.LookupReplicated(testkey1, 3)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// err = ts4.fids.fet.checkKey(testkey1, locs)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	ts1.fids.shutdownWait()
	ts2.fids.shutdownWait()
	ts3.fids.shutdownWait()
	ts4.fids.shutdownWait()
}
