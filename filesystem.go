package fidias

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/hexablock/blox/block"
	"github.com/hexablock/blox/filesystem"
	"github.com/hexablock/hexatype"
)

var (
	errFileExists        = errors.New("file exists")
	errFileOrDirNotFound = errors.New("file or directory not found")
)

// VersionedFileStore implements a storage mechanism for versioned file paths.
// Each file path may point many versions containing an alias and id
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
	//
	dev *RingDevice
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
		dev:    dev,
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

// Mkdir creates a new directory by submitting a log entry
func (fs *FileSystem) Mkdir(name string) error {
	// Get and check directory
	_, dver, tree, err := fs.getDir(name)
	if err != nil {
		return err
	}

	// Namespaced path
	nskey := append(fs.ns, []byte(name)...)
	// Get entry here for optimization and use these options for later calls
	entry, opts, err := fs.hexlog.NewEntry(nskey)
	if err != nil {
		return err
	}

	// Check if file exists
	ctx := context.Background()
	if _, err = fs.trans.GetPath(ctx, opts.PeerSet[0].Host(), name); err == nil {
		return errFileExists
	}

	// Create empty tree block
	ftree := block.NewTreeBlock(nil, fs.hasher)
	ftree.Hash()
	if _, err = fs.dev.SetBlock(ftree); err != nil && err != block.ErrBlockExists {
		return err
	}

	// Update file entry
	vers := NewVersionedFile(name)
	vers.AddVersion(&FileVersion{Alias: activeVersion, ID: ftree.ID()})
	val, err := vers.MarshalBinary()
	if err != nil {
		return err
	}
	entry.Data = append([]byte{OpFsSet}, val...)

	opts.WaitBallot = true
	if err = fs.hexlog.ProposeEntry(entry, opts); err != nil {
		return err
	}

	// We're at the root directory; nothing to do
	if dver == nil {
		return nil
	}

	// Update parent directory with current dir info
	tn := block.NewDirTreeNode(filepath.Base(name), ftree.ID())
	tree.AddNodes(tn)

	var tid []byte
	if tid, err = fs.dev.SetBlock(tree); err != nil {
		return err
	}

	if err = dver.UpdateVersion(activeVersion, tid); err != nil {
		return err
	}

	if entry, opts, err = fs.hexlog.NewEntryFrom(dver.entry); err != nil {
		return err
	}

	if val, err = dver.MarshalBinary(); err != nil {
		return err
	}

	entry.Data = append([]byte{OpFsSet}, val...)
	opts.WaitBallot = true

	return fs.hexlog.ProposeEntry(entry, opts)
}

// Create creates a new file
func (fs *FileSystem) Create(name string) (*File, error) {
	_, dir, tree, err := fs.getDir(name)
	if err != nil {
		return nil, err
	}

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

	opts.WaitBallot = true
	err = fs.hexlog.ProposeEntry(entry, opts)
	if err != nil {
		return nil, err
	}

	vers.entry = entry
	fh, err := fs.bfs.Create()
	if err != nil {
		return nil, err
	}

	return &File{
		versions: vers,
		tree:     tree,
		dver:     dir,
		BloxFile: fh,
		hexlog:   fs.hexlog,
		dev:      fs.dev,
	}, nil
}

// GetVersions gets the VersionedFile associated to the provided file name.
func (fs *FileSystem) GetVersions(name string) (*VersionedFile, error) {
	key := []byte(name)
	nskey := append(fs.ns, key...)

	locs, err := fs.dht.LookupReplicated(nskey, fs.hexlog.MinVotes())
	if err != nil {
		return nil, err
	}

	var (
		vers *VersionedFile
		ctx  = context.Background()
	)
	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	for _, loc := range locs {
		if vers, err = fs.trans.GetPath(ctx, loc.Host(), name); err == nil {
			break
		}
	}

	return vers, err
}

// Open opens the active version of the named file for reading. If successful,
// methods on the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
func (fs *FileSystem) Open(name string) (*File, error) {
	vers, err := fs.GetVersions(name)
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
	vers, err := fs.GetVersions(name)
	if err != nil {
		return nil, err
	}

	active := vers.Version()
	fh, err := fs.bfs.Stat(active.ID)
	if err != nil {
		return nil, err
	}

	return &File{versions: vers, BloxFile: fh.(*filesystem.BloxFile)}, nil
}

// getDir constructs the directory object for the given filename including the underlying
// BloxFile
func (fs *FileSystem) getDir(filename string) (string, *VersionedFile, *block.TreeBlock, error) {
	name := filepath.Dir(filename)
	if name == "." {
		return name, nil, nil, nil
	}

	vers, err := fs.GetVersions(name)
	if err != nil {
		return name, nil, nil, err
	}

	ver := vers.Version()
	var tree *block.TreeBlock
	if bytes.Compare(ver.ID, fs.hasher.ZeroHash()) == 0 {
		tree = block.NewTreeBlock(nil, fs.hasher)
	} else {
		bfh, err := fs.bfs.Open(ver.ID)
		if err != nil {
			return name, nil, nil, err
		}
		tree = bfh.Sys().(*block.TreeBlock)
	}

	return name, vers, tree, err
}
