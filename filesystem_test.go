package fidias

import (
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/hexablock/blox/block"
)

func TestFileSystem(t *testing.T) {

	ts1, err := newTestServer("127.0.0.1:49234", "127.0.0.1:11100")
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(100 * time.Millisecond)

	ts2, err := newTestServer("127.0.0.1:49235", "127.0.0.1:11101", "127.0.0.1:49234")
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(100 * time.Millisecond)

	ts3, err := newTestServer("127.0.0.1:49236", "127.0.0.1:11102", "127.0.0.1:49235")
	if err != nil {
		t.Fatal(err)
	}
	<-time.After(300 * time.Millisecond)

	defer ts1.cleanup()
	defer ts2.cleanup()
	defer ts3.cleanup()

	ffs, err := ts1.fids.fs.Create("foobar")
	if err != nil {
		t.Fatal(err)
	}

	if ffs.Flags() != os.O_WRONLY {
		t.Error("should have wronly flag")
	}

	tfh, _ := os.Open(testfile)
	defer tfh.Close()

	if _, err = io.Copy(ffs, tfh); err != nil {
		t.Error(err)
	}

	//t.Log("Closing ffs handle")
	if err = ffs.Close(); err != nil {
		t.Fatal(err)
	}

	rfh, err := ts2.fids.fs.Open("foobar")
	if err != nil {
		t.Fatal(err)
	}

	if rfh.Size() != ffs.Size() {
		t.Fatal("open size mismatch")
	}

	if rfh.Flags() != os.O_RDONLY {
		t.Error("should have rdonly flag")
	}

	active := rfh.versions.Version()
	t.Log(active.Text())

	tfile, _ := ioutil.TempFile(ts1.dir, "fidias-read-")
	if _, err = io.Copy(tfile, rfh); err != nil {
		t.Fatal(err)
	}
	tfile.Close()
	if err = rfh.Close(); err != nil {
		t.Error(err)
	}

	if err = ts1.fids.fs.Mkdir("user"); err != nil {
		t.Fatal(err)
	}

	fh, err := ts1.fids.fs.Create("user/foobar")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fh.Write([]byte("foofdfdlkfdflkdjfldkjfldkjfdl")); err != nil {
		t.Fatal(err)
	}
	if err = fh.Close(); err != nil {
		t.Fatal(err)
	}

	dir, err := ts1.fids.fs.Open("user")
	if err != nil {
		t.Fatal(err)
	}

	if !dir.IsDir() {
		t.Fatal("should be directory")
	}

	if err = ts1.fids.fs.Mkdir("fuber/uber"); err == nil {
		t.Fatal("should fail with='file not found' got='nil'")
	}

	if err = ts1.fids.fs.Mkdir("user/uber"); err != nil {
		t.Fatal(err)
	}

	if _, err = ts1.fids.fs.Open("user/uber"); err == nil {
		t.Fatal("should fail to open new dir")
	}

	d1, err := ts1.fids.fs.Open("user")
	if err != nil {
		t.Fatal(err)
	}

	if !d1.IsDir() {
		t.Fatal("should be directory")
	}

	tb := d1.Sys().(*block.TreeBlock)
	var c int
	tb.Iter(func(tn *block.TreeNode) error {
		c++
		return nil
	})
	if c != 2 {
		t.Fatal("should have 2 tree nodes")
	}

	ls, err := d1.Readdirnames(0)
	if err != nil {
		t.Fatal("failed to list files in dir", err)
	}

	if len(ls) != 2 {
		t.Fatal("should have 2 files got", len(ls))
	}

	ts1.fids.shutdownWait()
	ts2.fids.shutdownWait()
	ts3.fids.shutdownWait()
}
