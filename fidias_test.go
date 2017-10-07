package fidias

import (
	"bytes"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/hexablock/blox"
	"github.com/hexablock/blox/device"
	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexalog/store"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

func TestMain(m *testing.M) {
	log.SetLevel("INFO")
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	os.Exit(m.Run())
}

type testServer struct {
	dir    string
	hasher hexatype.Hasher

	bln    net.Listener
	btrans *blox.LocalTransport
	j      device.Journal
	bdev   *device.BlockDevice

	ln   net.Listener
	g    *grpc.Server
	c    *Config
	ps   hexaring.PeerStore
	r    *hexaring.Ring
	fids *Fidias

	rdev *RingDevice
}

func (ts *testServer) start() {
	go ts.g.Serve(ts.ln)
}

func (ts *testServer) cleanup() {
	os.RemoveAll(ts.dir)
}

func (ts *testServer) setupBlox() {
	ts.dir, _ = ioutil.TempDir("", "fidias")

	rdev, _ := device.NewFileRawDevice(ts.dir, ts.hasher)
	dev := device.NewBlockDevice(ts.j, rdev)

	opts := blox.DefaultNetClientOptions(ts.hasher)
	remote := blox.NewNetTransport(ts.bln, opts)
	trans := blox.NewLocalTransport(ts.bln.Addr().String(), remote)
	trans.Register(dev)

	ts.btrans = trans
	ts.bdev = dev
}

func newTestServer(addr, bloxAddr string, peers ...string) (*testServer, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	bln, err := net.Listen("tcp", bloxAddr)
	if err != nil {
		return nil, err
	}

	ts := &testServer{
		j:   device.NewInmemJournal(),
		bln: bln,
		ln:  ln,
		g:   grpc.NewServer(),
		c:   DefaultConfig(addr),
		ps:  hexaring.NewInMemPeerStore(),
	}
	ts.hasher = ts.c.Hexalog.Hasher
	ts.setupBlox()

	// Ring
	ts.c.Ring.StabilizeMin = time.Duration(15 * time.Millisecond)
	ts.c.Ring.StabilizeMax = time.Duration(45 * time.Millisecond)
	ts.c.Ring.Meta = chord.Meta{"key": []byte("test"), "blox": []byte(bloxAddr)}
	for _, p := range peers {
		ts.ps.AddPeer(p)
	}

	chordTrans := chord.NewGRPCTransport(3*time.Second, 10*time.Second)
	ts.r = hexaring.New(ts.c.Ring, ts.ps, chordTrans)
	ts.r.RegisterServer(ts.g)

	// Stores
	idx := store.NewInMemIndexStore()
	ss := &store.InMemStableStore{}
	es := store.NewInMemEntryStore()
	ls := hexalog.NewLogStore(es, idx, ts.c.Hexalog.Hasher)
	fsm := NewFSM(ts.c.KeyValueNamespace, ts.c.FileSystemNamespace)

	// Hexalog
	logNet := hexalog.NewNetTransport(3*time.Second, 3*time.Second)
	hexalog.RegisterHexalogRPCServer(ts.g, logNet)

	hexlog, err := NewHexalog(ts.c, ls, ss, fsm, logNet)
	if err != nil {
		return nil, err
	}

	ts.hasher = ts.c.Hasher()

	// Fetcher
	fet := NewFetcher(idx, es, ts.c.Hexalog.Votes, ts.c.RelocateBufSize, ts.c.Hasher())
	fet.RegisterBlockTransport(ts.btrans)
	// Relocator
	rel := NewRelocator(int64(ts.c.Hexalog.Votes), ts.c.Hasher())
	rel.RegisterBlockJournal(ts.j)
	rel.RegisterKeylogIndex(idx)
	// Key-value
	keyvs := NewKeyvs(ts.c.KeyValueNamespace, hexlog, fsm)

	// Fidias
	fidTrans := NewNetTransport(fsm, idx, ts.j, 30*time.Second, 3*time.Second, ts.c.Hexalog.Votes, ts.c.Hasher())
	RegisterFidiasRPCServer(ts.g, fidTrans)

	ts.rdev = NewRingDevice(2, ts.c.Hasher(), ts.bdev, ts.btrans)
	ts.fids = New(ts.c, hexlog, fsm, rel, fet, keyvs, ts.rdev, fidTrans)

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
	ts1, err := newTestServer("127.0.0.1:61234", "127.0.0.1:62100")
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(100 * time.Millisecond)

	ts2, err := newTestServer("127.0.0.1:61235", "127.0.0.1:62101", "127.0.0.1:61234")
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(100 * time.Millisecond)

	ts3, err := newTestServer("127.0.0.1:61236", "127.0.0.1:62102", "127.0.0.1:61235")
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(300 * time.Millisecond)

	defer ts1.cleanup()
	defer ts2.cleanup()
	defer ts3.cleanup()

	//
	// Hexalog
	//
	testkey1 := []byte("testkey1")
	_, _, err = ts1.fids.hexlog.NewEntry(testkey1)
	if err != nil {
		t.Fatal(err)
	}

	entry, opt, err := ts2.fids.hexlog.NewEntry(testkey1)
	if err != nil {
		t.Fatal(err)
	}
	//&hexatype.RequestOptions{PeerSet: remeta.PeerSet}
	opt.WaitBallot = true
	opt.WaitApply = true
	opt.WaitApplyTimeout = 1000
	err = ts1.fids.hexlog.ProposeEntry(entry, opt)
	if err != nil {
		t.Fatal(err)
	}

	// if err = ballot.Wait(); err != nil {
	// 	t.Fatal(err)
	// }

	// fe := ballot.Future()
	// if _, err = fe.Wait(1 * time.Second); err != nil {
	// 	t.Fatal(err)
	// }

	id := entry.Hash(ts2.c.Hasher().New())
	if _, _, err = ts2.fids.hexlog.GetEntry(entry.Key, id); err != nil {
		t.Fatal(err)
	}

	ki, err := ts3.fids.hexlog.trans.store.GetKey(entry.Key)
	if err != nil {
		t.Fatal(err)
	}

	kidx := ki.GetIndex()
	if bytes.Compare(kidx.Last(), id) != 0 {
		t.Fatal("id mismatch")
	}

	// New node joining
	<-time.After(10 * time.Millisecond)
	ts4, err := newTestServer("127.0.0.1:61237", "127.0.0.1:62103", "127.0.0.1:61234")
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(200 * time.Millisecond)

	defer ts4.cleanup()

	_, meta, err := ts3.fids.keyvs.SetKey([]byte("test"), []byte("val"))
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.PeerSet) != 3 {
		t.Fatal("should have 3 peers")
	}

	// if _, err = fut.Wait(2 * time.Second); err != nil {
	// 	t.Fatal(err)
	// }

	//
	// Keyvs
	//
	gk, _, err := ts3.fids.keyvs.GetKey([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}
	if gk == nil {
		t.Fatal("keyvs get key should not be nil")
	}
	if string(gk.Value) != "val" {
		t.Fatal("keyvs value mismatch")
	}

	if _, _, err = ts2.fids.keyvs.RemoveKey([]byte("test")); err != nil {
		t.Fatal("failed to removed", err)
	}

	if _, _, err = ts3.fids.keyvs.GetKey([]byte("test")); err != hexatype.ErrKeyNotFound {
		t.Fatalf("keyvs should fail with='%v' got='%v'", hexatype.ErrKeyNotFound, err)
	}

	t.Logf("%+v", ts3.fids.Status())

	ts1.fids.shutdownWait()
	ts2.fids.shutdownWait()
	ts3.fids.shutdownWait()
	ts4.fids.shutdownWait()
}
