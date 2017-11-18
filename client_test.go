package fidias

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"

	"github.com/hexablock/blox"
)

func Test_Client(t *testing.T) {

	fid0, err := newTestFidias("127.0.0.1:42000", "127.0.0.1:17080", "127.0.0.1", 49950)
	if err != nil {
		t.Fatal(err)
	}
	// node 2
	fid1, err := newTestFidias("127.0.0.1:42001", "127.0.0.1:17081", "127.0.0.1", 49951)
	if err != nil {
		t.Fatal(err)
	}
	if err = fid1.Join([]string{"127.0.0.1:49950"}); err != nil {
		t.Fatal(err)
	}
	// node 3
	fid2, err := newTestFidias("127.0.0.1:42002", "127.0.0.1:17082", "127.0.0.1", 49952)
	if err != nil {
		t.Fatal(err)
	}
	if err = fid2.Join([]string{"127.0.0.1:49950"}); err != nil {
		t.Fatal(err)
	}

	<-time.After(1 * time.Second)

	// BEGIN CLIENT TEST

	conf := DefaultConfig()
	//conf.Peers = []string{"127.0.0.1:42000"}
	conf.Phi.Hexalog.AdvertiseHost = "127.0.0.1:17081"
	client, err := NewClient(conf)
	if err != nil {
		t.Fatal(err)
	}

	blx := blox.NewBlox(client.dev)
	rd := ioutil.NopCloser(bytes.NewBuffer([]byte("qaswerfsasdfghjkoiuytrewerfghnfewqwedfvbvdwqwsdsasxzzaqwertyukoioiuytrewghgfbvcxzawwertg")))
	if _, err = blx.WriteIndex(rd, 2); err != nil {
		t.Fatal(err)
	}

	kvstore := client.KV()

	wo := DefaultWriteOptions()
	kv := NewKVPair([]byte("some-test-key"), []byte("value"))
	if _, _, err = kvstore.Set(kv, wo); err != nil {
		t.Fatal(err)
	}

	if kv, _, err = kvstore.Get([]byte("some-test-key"), nil); err != nil {
		t.Fatal(err)
	}
	if string(kv.Value) != "value" {
		t.Fatal("wrong value")
	}

	if _, _, err = kvstore.CASet(kv, kv.Modification, wo); err != nil {
		t.Fatal(err)
	}

	if _, err = kvstore.Remove(kv.Key, wo); err != nil {
		t.Fatal(err)
	}

	//	<-time.After(2 * time.Second)
	if _, _, err = kvstore.Get(kv.Key, &ReadOptions{}); err == nil {
		t.Fatal("should fail")
	}

	rkv := NewKVPair([]byte("key-remove"), []byte("remove"))
	rkvMod, _, err := kvstore.Set(rkv, wo)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = kvstore.CARemove(rkv.Key, rkvMod.Modification, wo); err != nil {
		t.Fatal(err)
	}

	kv1 := NewKVPair([]byte("key-remove1"), []byte("value"))
	if _, _, err = kvstore.Set(kv1, wo); err != nil {
		t.Fatal(err)
	}

	//<-time.After(100 * time.Millisecond)
	nodes, err := client.dht.Lookup([]byte("kv/key-remove1"))
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) < 1 {
		t.Error("lookup has no nodes")
	}

	fid0.Shutdown()
	fid1.Shutdown()
	fid2.Shutdown()

}
