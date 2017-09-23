package fidias

import (
	"bytes"
	"testing"
)

func TestVersioned(t *testing.T) {
	vkey := NewVersionedFile("key")
	version := &FileVersion{Alias: "head", ID: []byte("12345678901234567890123456789012")}
	if err := vkey.AddVersion(version); err != nil {
		t.Fatal(err)
	}

	if err := vkey.AddVersion(version); err != ErrVersionExists {
		t.Fatal("should fail with", ErrVersionExists, err)
	}

	version.ID = []byte("qplaplapla")
	if err := vkey.UpdateVersion(version); err != nil {
		t.Fatal(err)
	}

	if len(vkey.versions) != 1 {
		t.Fatalf("should have 1")
	}

	v2 := &FileVersion{Alias: "name", ID: []byte("123456765434567654456765y")}
	if err := vkey.UpdateVersion(v2); err != ErrVersionNotFound {
		t.Fatal("should fail with", ErrVersionNotFound, err)
	}
	vkey.AddVersion(v2)

	b, _ := vkey.MarshalBinary()

	vkey1 := &VersionedFile{}

	if err := vkey1.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}

	for k, v := range vkey.versions {
		v1, ok := vkey1.versions[k]
		if !ok {
			t.Fatal("key not found", k)
		}

		if v1.Alias != v.Alias {
			t.Fatal("name mismatch")
		}
		if bytes.Compare(v1.ID, v.ID) != 0 {
			t.Fatal("id mismatch")
		}
	}

}
