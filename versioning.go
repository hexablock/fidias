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

// Text returns the text string representation of the file version
func (ver *FileVersion) Text() string {
	return hex.EncodeToString(ver.ID) + " " + ver.Alias
}

type VersionedFile struct {
	// Name i.e. full path of the file
	name string
	// Alias to version hash map
	versions map[string]*FileVersion
	// Entry associate to this view
	entry *hexatype.Entry
}

func NewVersionedFile(name string) *VersionedFile {
	return &VersionedFile{
		name:     name,
		versions: make(map[string]*FileVersion),
	}
}

func (f *VersionedFile) UpdateVersion(version *FileVersion) error {

	if ver, ok := f.versions[version.Alias]; ok {
		f.versions[version.Alias] = ver
		return nil
	}

	return ErrVersionNotFound
}

// AddVersion adds a new version
func (f *VersionedFile) AddVersion(version *FileVersion) error {
	if _, ok := f.versions[version.Alias]; !ok {
		f.versions[version.Alias] = version
		return nil
	}

	return ErrVersionExists
}

// Version returns the active version
func (f *VersionedFile) Version() *FileVersion {
	ver, _ := f.versions[activeVersion]
	return ver
}

func (f *VersionedFile) GetVersion(alias string) (*FileVersion, error) {
	if val, ok := f.versions[alias]; ok {
		return val, nil
	}

	return nil, ErrVersionNotFound
}

func (f *VersionedFile) String() string {
	out := make([]string, len(f.versions))
	var i int
	for _, v := range f.versions {
		out[i] = v.Text()
		i++
	}
	return strings.Join(out, "\n")
}

// MarshalBinary marshals the version into a byte slice.  It does not include
// the key and entry
func (f *VersionedFile) MarshalBinary() ([]byte, error) {
	return []byte(f.String()), nil
}

// UnmarshalBinary unmarshal the byte slice into Versioned.  It will not include
// the key and entry
func (f *VersionedFile) UnmarshalBinary(b []byte) error {
	arr := strings.Split(string(b), "\n")

	if f.versions == nil {
		f.versions = make(map[string]*FileVersion)
	}

	for _, a := range arr {
		p := strings.Split(a, " ")
		if len(p) != 2 {
			return fmt.Errorf("invalid Versioned data")
		}

		ver := &FileVersion{Alias: p[1]}
		id, err := hex.DecodeString(p[0])
		if err != nil {
			return err
		}
		ver.ID = id
		f.versions[ver.Alias] = ver
	}

	return nil
}
