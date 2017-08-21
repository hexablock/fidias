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

	_, _, err = ts1.fids.NewEntry([]byte("testkey1"), 2)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = ts2.fids.NewEntry([]byte("testkey1"), 2)
	if err != nil {
		t.Fatal(err)
	}

}
