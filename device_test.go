package fidias

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/hexablock/blox/block"
	"github.com/hexablock/blox/filesystem"
)

var (
	testfile = "./test-data/Crypto101.pdf"
)

func TestRingDevice(t *testing.T) {
	ts1, err := newTestServer("127.0.0.1:33321", "127.0.0.1:55443")
	if err != nil {
		t.Fatal(err)
	}
	ts1.setupBlox()
	defer ts1.cleanup()
	<-time.After(100 * time.Millisecond)

	ts2, err := newTestServer("127.0.0.1:33322", "127.0.0.1:55444", "127.0.0.1:33321")
	if err != nil {
		t.Fatal(err)
	}
	ts2.setupBlox()
	defer ts2.cleanup()
	<-time.After(100 * time.Millisecond)

	ts3, err := newTestServer("127.0.0.1:33323", "127.0.0.1:55445", "127.0.0.1:33322")
	if err != nil {
		t.Fatal(err)
	}
	ts3.setupBlox()
	defer ts3.cleanup()
	<-time.After(300 * time.Millisecond)

	ringDev := NewRingDevice(3, ts1.hasher, ts1.btrans)
	ringDev.Register(ts1.r)
	fs := filesystem.NewBloxFS(ringDev)

	if ringDev.locator == nil {
		t.Fatal("ring should not be nil")
	}

	nf, err := fs.Create()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(fs.Name(), nf.BlockSize())
	//
	fh, err := os.Open(testfile)
	if err != nil {
		t.Fatal(err)
	}

	defer fh.Close()

	// n := (1024 * 1024) + 2048
	// b := make([]byte, n)
	// rn, err := fh.Read(b)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// if rn != n {
	// 	t.Error("did not read all")
	// }

	// rn, err = nf.Write(b)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// if rn != n {
	// 	t.Error("did not write all")
	// }

	//
	if _, err = io.Copy(nf, fh); err != nil {
		t.Fatal(err)
	}
	if err = nf.Close(); err != nil {
		t.Fatal(err)
	}

	if nf.Size() == 0 {
		t.Error("size should not be 0")
	}

	stat, _ := fh.Stat()
	if nf.Size() != stat.Size() {
		t.Fatal("size msmatch")
	}

	s := nf.Sys()
	idx := s.(*block.IndexBlock)

	if idx.BlockCount() != 16 {
		t.Fatal("should have 16 blocks")
	}
}
