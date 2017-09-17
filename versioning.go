package fidias

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/hexablock/hexatype"
)

const activeVersion = "active"

var (
	ErrVersionNotFound = errors.New("version not found")
	ErrVersionExists   = errors.New("version exists")
)

// Version represents a version of a given key.  It contains the version
// name and the id it points to
type Version struct {
	Name string
	ID   []byte
}

func (ver *Version) String() string {
	return hex.EncodeToString(ver.ID) + " " + ver.Name
}

type Versioned struct {
	key      []byte
	versions map[string]*Version
	// Entry associate to this view
	entry *hexatype.Entry
}

func NewVersioned(key []byte) *Versioned {
	return &Versioned{
		key:      key,
		versions: make(map[string]*Version),
	}
}

func (f *Versioned) Version() *Version {
	ver, _ := f.versions[activeVersion]
	return ver
}

func (f *Versioned) UpdateVersion(version *Version) error {

	if ver, ok := f.versions[version.Name]; ok {
		f.versions[version.Name] = ver
		return nil
	}

	return ErrVersionNotFound
}

// AddVersion adds a new version
func (f *Versioned) AddVersion(version *Version) error {
	if _, ok := f.versions[version.Name]; !ok {
		f.versions[version.Name] = version
		return nil
	}

	return ErrVersionExists
}

func (f *Versioned) String() string {
	out := make([]string, len(f.versions))
	var i int
	for _, v := range f.versions {
		out[i] = v.String()
		i++
	}
	return strings.Join(out, "\n")
}

// MarshalBinary marshals the version into a byte slice.  It does not include
// the key and entry
func (f *Versioned) MarshalBinary() ([]byte, error) {
	return []byte(f.String()), nil
}

// UnmarshalBinary unmarshal the byte slice into Versioned.  It will not include
// the key and entry
func (f *Versioned) UnmarshalBinary(b []byte) error {
	arr := strings.Split(string(b), "\n")

	if f.versions == nil {
		f.versions = make(map[string]*Version)
	}

	for _, a := range arr {
		p := strings.Split(a, " ")
		if len(p) != 2 {
			return fmt.Errorf("invalid Versioned data")
		}

		ver := &Version{Name: p[1]}
		id, err := hex.DecodeString(p[0])
		if err != nil {
			return err
		}
		ver.ID = id
		f.versions[ver.Name] = ver
	}

	return nil
}
