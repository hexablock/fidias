package fidias

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hexablock/hexalog"
)

const activeVersion = "active"

var (
	// ErrVersionNotFound is used when a file version is not found
	ErrVersionNotFound = errors.New("version not found")
	// ErrVersionExists is used when a new version being created has
	// the same name as an already existing one
	ErrVersionExists = errors.New("version exists")
)

// Text returns the text string representation of the file version
func (ver *FileVersion) Text() string {
	return hex.EncodeToString(ver.ID) + " " + ver.Alias
}

// MarshalJSON marshals a file version accounting for hash ids
func (ver *FileVersion) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Alias string
		ID    string
	}{
		Alias: ver.Alias,
		ID:    hex.EncodeToString(ver.ID),
	})
}

// VersionedFile contains all known versions for a given file and the hexalog
// entry associated with the view instance
type VersionedFile struct {
	// Full path of the file
	name string
	// Alias to version hash map
	versions map[string]*FileVersion
	// Entry associate to this view
	entry *hexalog.Entry
}

// NewVersionedFile instantiates a new VersionedFile with the given name
func NewVersionedFile(name string) *VersionedFile {
	return &VersionedFile{
		name:     name,
		versions: make(map[string]*FileVersion),
	}
}

// UpdateVersion updates a version by the alias. It returns an
// ErrVersionNotFound if the alias is not found,
func (f *VersionedFile) UpdateVersion(alias string, id []byte) error {

	if ver, ok := f.versions[alias]; ok {
		ver.ID = id
		f.versions[alias] = ver
		return nil
	}

	return ErrVersionNotFound
}

// AddVersion adds a new version of the file.  It returns an ErrVersionExists
// if the alias for the given version already exists.
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

// GetVersion gets a version by the given alias.
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

// MarshalJSON marshals the VersionedFile as json with the wanted private
// fields
func (f *VersionedFile) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name     string
		Versions map[string]*FileVersion
	}{f.name, f.versions})
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
