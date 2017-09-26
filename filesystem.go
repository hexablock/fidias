package fidias

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/hexablock/blox/block"
	"github.com/hexablock/blox/filesystem"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexatype"
)

var (
	errFileExists   = errors.New("file exists")
	errFileNotFound = errors.New("file not found")
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
		return errFileNotFound
	}

	nskey := append(fs.ns, []byte(name)...)

	entry, opts, err := fs.hexlog.NewEntry(nskey)
	if err != nil {
		return err
	}

	if _, err = fs.trans.GetPath(context.Background(), opts.PeerSet[0].Host(), name); err == nil {
		return errFileExists
	}

	// Update file entry
	vers := NewVersionedFile(name)
	vers.AddVersion(&FileVersion{Alias: activeVersion, ID: fs.hasher.ZeroHash()})
	val, err := vers.MarshalBinary()
	if err != nil {
		return err
	}
	entry.Data = append([]byte{OpFsSet}, val...)
	if _, err = fs.submitEntry(entry, opts); err != nil {
		return err
	}

	// Update parent directory with current dir info
	if dver != nil {
		tn := block.NewDirTreeNode(filepath.Base(name), fs.hasher.ZeroHash())
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
		if _, err = fs.submitEntry(entry, opts); err != nil {
			return err
		}

	}

	return err
}

func (fs *FileSystem) getDir(filename string) (string, *VersionedFile, *block.TreeBlock, error) {
	name := filepath.Dir(filename)
	if name == "." {
		return name, nil, nil, nil
	}

	nskey := append(fs.ns, []byte(name)...)
	locs, err := fs.dht.LookupReplicated(nskey, fs.hexlog.MinVotes())
	if err != nil {
		return name, nil, nil, err
	}

	vers, err := fs.trans.GetPath(context.Background(), locs[0].Host(), name)
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

	fent, err := fs.submitEntry(entry, opts)
	if err != nil {
		return nil, err
	}

	vers.entry = fent.Entry
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

func (fs *FileSystem) submitEntry(entry *hexatype.Entry, opts *hexatype.RequestOptions) (*hexalog.FutureEntry, error) {
	ballot, err := fs.hexlog.ProposeEntry(entry, opts)
	if err != nil {
		return nil, err
	}
	if err = ballot.Wait(); err == nil {
		return ballot.Future(), nil

	}
	return nil, err
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
	//log.Printf("ACTIVE %s", active.Text())
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
