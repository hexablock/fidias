package fidias

import (
	"context"
	"errors"
	"os"

	"github.com/hexablock/blox/filesystem"
	"github.com/hexablock/hexatype"
)

var (
	errFileExists   = errors.New("file exists")
	errFileNotFound = errors.New("file not found")
)

type VersionedFileStore interface {
	GetPath(name string) (*VersionedFile, error)
}

// FileSystem represents a fidias filesystem
type FileSystem struct {
	// Hexalog namespace
	ns []byte

	hasher hexatype.Hasher
	// DHT ring
	dht DHT
	// Hexalog fs write operations
	hexlog *Hexalog
	// Content addressable file-system
	bfs *filesystem.BloxFS
	// Transport for read operations
	trans *localFileSystemTransport
}

// NewFileSystem inits a new FileSystem instance.  There can be as many instances needed.
// namespace is used to prefix all keys.
func NewFileSystem(host, namespace string, dev *RingDevice, hexlog *Hexalog, verfs VersionedFileStore) *FileSystem {
	trans := &localFileSystemTransport{
		host:  host,
		local: verfs,
	}

	fs := &FileSystem{
		ns:     []byte(namespace),
		hexlog: hexlog,
		bfs:    filesystem.NewBloxFS(dev),
		trans:  trans,
	}

	fs.hasher = fs.bfs.Hasher()
	return fs
}

// RegisterTransport registers a network transport for the filesystem used to get
// remote paths
func (fs *FileSystem) RegisterTransport(remote FileSystemTransport) {
	fs.trans.remote = remote
}

// RegisterDHT registers the DHT for lookups
func (fs *FileSystem) RegisterDHT(dht DHT) {
	fs.dht = dht
}

// Create creates a new file
func (fs *FileSystem) Create(name string) (*File, error) {
	//key := []byte(name)
	nskey := append(fs.ns, []byte(name)...)

	entry, opts, err := fs.hexlog.NewEntry(nskey)
	if err != nil {
		return nil, err
	}

	if _, err = fs.trans.GetPath(context.Background(), opts.PeerSet[0].Host(), name); err == nil {
		return nil, errFileExists
	}

	// Create the first version being the zero hash
	vers := NewVersionedFile(name)
	vers.AddVersion(&FileVersion{Alias: activeVersion, ID: fs.hasher.ZeroHash()})

	val, err := vers.MarshalBinary()
	if err != nil {
		return nil, err
	}

	entry.Data = append([]byte{OpFsSet}, val...)

	ballot, err := fs.hexlog.ProposeEntry(entry, opts)
	if err != nil {
		return nil, err
	}
	if err = ballot.Wait(); err != nil {
		return nil, err
	}
	fent := ballot.Future()

	vers.entry = fent.Entry
	fh, err := fs.bfs.Create()
	if err != nil {
		return nil, err
	}

	return &File{versions: vers, BloxFile: fh, hexlog: fs.hexlog}, nil
}

// Open opens the active version of the named file for reading. If successful,
// methods on the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
func (fs *FileSystem) Open(name string) (*File, error) {
	key := []byte(name)
	nskey := append(fs.ns, key...)

	locs, err := fs.dht.LookupReplicated(nskey, fs.hexlog.MinVotes())
	if err != nil {
		return nil, err
	}

	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	ctx := context.Background()

	vers, err := fs.trans.GetPath(ctx, locs[0].Host(), name)
	if err != nil {
		return nil, err
	}

	active := vers.Version()
	fh, err := fs.bfs.Open(active.ID)
	if err != nil {
		return nil, err
	}

	file := &File{versions: vers, BloxFile: fh}
	return file, nil
}

// Stat performs a stat call on the file returning a standard os.FileInfo object
func (fs *FileSystem) Stat(name string) (os.FileInfo, error) {
	key := []byte(name)
	locs, err := fs.dht.LookupReplicated(key, fs.hexlog.MinVotes())
	if err != nil {
		return nil, err
	}

	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	ctx := context.Background()
	vers, err := fs.trans.GetPath(ctx, locs[0].Host(), name)
	if err != nil {
		return nil, err
	}

	active := vers.Version()
	fh, err := fs.bfs.Stat(active.ID)
	if err != nil {
		return nil, err
	}
	bf := fh.(*filesystem.BloxFile)

	return &File{versions: vers, BloxFile: bf}, nil
}
