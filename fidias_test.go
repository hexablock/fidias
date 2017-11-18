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
	c := DefaultConfig()

	conf := c.Phi
	conf.Memberlist = testMemberlistConfig(klpAddr, host, port)
	conf.DHT = kelips.DefaultConfig(klpAddr)
	conf.DHT.Meta["hexalog"] = httpAddr
	conf.Hexalog = hexalog.DefaultConfig(httpAddr)
	conf.Hexalog.Votes = 2
	conf.DataDir, _ = ioutil.TempDir("/tmp", "fid-")
	conf.SetHashFunc(sha256.New)

	return Create(c)
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

	if _, err = blx.WriteIndex(rd, 2); err != nil {
		t.Fatal(err)
	}

	kvs := fid1.KVS()
	wo := DefaultWriteOptions()
	kv := NewKVPair([]byte("key"), []byte("value"))
	if _, _, err = kvs.Set(kv, wo); err != nil {
		t.Fatal(err)
	}

	kvp, _, err := kvs.Get([]byte("key"), &ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(kvp.Value) != "value" {
		t.Fatal("wrong value")
	}

	kv = NewKVPair([]byte("vault/0.8"), []byte("value"))
	nkv, _, err := kvs.Set(kv, wo)
	if err != nil {
		t.Fatal(err)
	}

	<-time.After(200 * time.Millisecond)
	ls, _, err := kvs.List([]byte("vault"), &ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ls) != 1 {
		t.Fatal("should have a 1 item")
	}

	k := NewKVPair([]byte("vault/0.8"), []byte("newvalue"))
	_, _, err = kvs.CASet(k, nkv.Modification, wo)
	if err != nil {
		t.Fatal(err)
	}

	vkv, _, err := kvs.Get([]byte("vault/0.8"), &ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if _, err = kvs.Remove([]byte("key"), wo); err != nil {
		t.Fatal(err)
	}

	<-time.After(50 * time.Millisecond)
	if _, _, err = kvs.Get([]byte("key"), &ReadOptions{}); err == nil {
		t.Error("should fail")
	}

	if _, err = kvs.CARemove([]byte("vault/0.8"), vkv.Modification, wo); err != nil {
		t.Fatal(err)
	}

	<-time.After(20 * time.Millisecond)
	if _, _, err = kvs.Get([]byte("vault/0.8"), &ReadOptions{}); err == nil {
		t.Fatal("should fail")
	}
}
