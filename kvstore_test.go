package fidias

import (
	"testing"
)

func Test_InmemKVStore(t *testing.T) {
	kvs := NewInmemKVStore()

	top := NewKVPair([]byte("top"), []byte("value"))
	dir1 := NewKVPair([]byte("top/dir1"), []byte("value"))

	kvs.Set(NewKVPair([]byte("top/dir1/file1"), []byte("value")))
	kvs.Set(NewKVPair([]byte("top/dir1/file2"), []byte("value")))
	kvs.Set(NewKVPair([]byte("top/dir2"), []byte("value")))
	kvs.Set(NewKVPair([]byte("top/dir2/file2"), []byte("value")))
	kvs.Set(dir1)
	kvs.Set(top)

	var c int
	kvs.Iter([]byte("top/dir1"), true, func(kvp *KVPair) bool {
		t.Logf("%s", kvp.Key)
		c++
		return true
	})
	if c != 3 {
		t.Error("should have 3 keys")
	}

	kvs.Iter([]byte("top/"), true, func(kvp *KVPair) bool {
		if string(kvp.Key) != "top/dir1" {
			t.Error("should be top/dir1")
		}
		return false
	})

	kvs.Iter([]byte("top/dir1/"), false, func(kvp *KVPair) bool {
		t.Log(string(kvp.Key))
		return true
	})

}

func Test_parseDirBytes(t *testing.T) {
	dir, _ := parseDirBytes([]byte("dir/subdir/key"))
	if string(dir) != "dir/subdir" {
		t.Fatalf("have=%s want=dir/subdir", dir)
	}

	dir, _ = parseDirBytes([]byte("key"))
	if string(dir) != "key" {
		t.Fatalf("have=%s want=key", dir)
	}
}

func parseDirBytes(key []byte) ([]byte, bool) {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '/' {
			return key[:i], true
		}
	}
	return key, false
}
