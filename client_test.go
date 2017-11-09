package fidias

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"
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

	conf := DefaultConfig()
	conf.Peers = []string{"127.0.0.1:42000"}
	client, err := NewClient(conf)
	if err != nil {
		t.Fatal(err)
	}

	kvs := client.KVS()
	wo := DefaultWriteOptions()
	kv := NewKVPair([]byte("some-test-key"), []byte("value"))
	if _, _, err = kvs.Set(kv, wo); err != nil {
		t.Fatal(err)
	}

	kvp, _, err := kvs.Get([]byte("some-test-key"), &ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(kvp.Key) != "some-test-key" {
		t.Fatal("key mismatch")
	}

	blx := client.Blox()
	rd := ioutil.NopCloser(bytes.NewBuffer([]byte("qaswerfsasdfghjkoiuytrewerfghnfewqwedfvbvdwqwsdsasxzzaqwertyukoioiuytrewghgfbvcxzawwertg")))
	if err = blx.WriteIndex(rd); err != nil {
		t.Fatal(err)
	}

	fid0.Shutdown()
	fid1.Shutdown()
	fid2.Shutdown()

}
