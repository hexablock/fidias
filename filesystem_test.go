package fidias

import (
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"
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

	ts1.fids.shutdownWait()
	ts2.fids.shutdownWait()
	ts3.fids.shutdownWait()
}
