package fidias

import (
	"crypto/sha256"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/hexablock/blox"
	"github.com/hexablock/go-kelips"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/log"
)

func TestMain(t *testing.M) {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
	log.SetLevel("DEBUG")
	os.Exit(t.Run())
}

func testMemberlistConfig(klpAddr, host string, port int) *memberlist.Config {
	conf := memberlist.DefaultLocalConfig()
	//conf.Name = fmt.Sprintf("%s:%d", host, port)
	conf.Name = klpAddr

	conf.GossipInterval = 50 * time.Millisecond
	conf.ProbeInterval = 500 * time.Millisecond
	conf.ProbeTimeout = 250 * time.Millisecond
	conf.SuspicionMult = 1

	conf.AdvertiseAddr = host
	conf.AdvertisePort = port
	conf.BindAddr = host
	conf.BindPort = port

	return conf
}

func newTestFidias(klpAddr, httpAddr, host string, port int) (*Fidias, error) {

	conf := DefaultConfig()
	conf.Memberlist = testMemberlistConfig(klpAddr, host, port)
	conf.DHT = kelips.DefaultConfig(klpAddr)
	conf.DHT.Meta["hexalog"] = httpAddr
	conf.Hexalog = hexalog.DefaultConfig(httpAddr)
	conf.Hexalog.Votes = 2
	conf.DataDir, _ = ioutil.TempDir("/tmp", "fid-")
	conf.SetHashFunc(sha256.New)
	return Create(conf)
}

func Test_Fidias(t *testing.T) {
	// node 1
	fid0, err := newTestFidias("127.0.0.1:41000", "127.0.0.1:18080", "127.0.0.1", 44550)
	if err != nil {
		t.Fatal(err)
	}
	// node 2
	fid1, err := newTestFidias("127.0.0.1:41001", "127.0.0.1:18081", "127.0.0.1", 44551)
	if err != nil {
		t.Fatal(err)
	}
	if err = fid1.Join([]string{"127.0.0.1:44550"}); err != nil {
		t.Fatal(err)
	}
	// node 3
	fid2, err := newTestFidias("127.0.0.1:41002", "127.0.0.1:18082", "127.0.0.1", 44552)
	if err != nil {
		t.Fatal(err)
	}
	if err = fid2.Join([]string{"127.0.0.1:44550"}); err != nil {
		t.Fatal(err)
	}

	<-time.After(2 * time.Second)

	dht0 := fid0.DHT()
	if err = dht0.Insert([]byte("testkey"), kelips.NewTupleHost("127.0.0.1:41001")); err != nil {
		t.Fatal(err)
	}

	dev := fid1.BlockDevice()
	blx := blox.NewBlox(dev)
	rd, _ := os.Open("./pool.go")
	defer rd.Close()

	if err = blx.WriteIndex(rd); err != nil {
		t.Fatal(err)
	}

	kvs := fid1.KVS()
	wo := DefaultWriteOptions()
	kv := NewKVPair([]byte("key"), []byte("value"))
	if _, _, err = kvs.Set(kv, wo); err != nil {
		t.Fatal(err)
	}

}
