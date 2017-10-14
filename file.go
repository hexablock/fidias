package fidias

import (
	"os"
	"path/filepath"

	"github.com/hexablock/blox/block"
	"github.com/hexablock/blox/filesystem"
	"github.com/hexablock/hexalog"
)

// File is a fidias file representing a standard OS file type interface.
// It contains methods to perform native go file type operations
type File struct {
	*filesystem.BloxFile

	dev *RingDevice

	dver *VersionedFile
	tree *block.TreeBlock

	versions *VersionedFile // file versions
	hexlog   *Hexalog
}

// Name returns the absolute path name of the file
func (file *File) Name() string {
	return file.versions.name
}

// Versions returns the underlying VersionedFile instance
func (file *File) Versions() *VersionedFile {
	return file.versions
}

// Close closes the underlying BloxFile and updates hexalog with the new
// hash entries
func (file *File) Close() error {
	err := file.BloxFile.Close()
	// Continue to update hexalog if the block exists
	if err != nil && err != block.ErrBlockExists {
		return err
	}

	// Nothing to do if not writing
	if file.Flags() != os.O_WRONLY {
		return nil
	}

	// Update file version if we are writing
	idx := file.BloxFile.Sys().(*block.IndexBlock)
	ver := file.versions.Version()
	ver.ID = idx.ID()
	if err = file.versions.UpdateVersion(ver.Alias, ver.ID); err != nil {
		return err
	}

	edata, err := file.versions.MarshalBinary()
	if err != nil {
		return err
	}

	// Update File and versions
	var (
		entry *hexalog.Entry
		opts  *hexalog.RequestOptions
	)
	entry, opts, err = file.hexlog.NewEntryFrom(file.versions.entry)
	if err != nil {
		return err
	}
	entry.Data = append([]byte{OpFsSet}, edata...)

	opts.WaitBallot = true
	err = file.hexlog.ProposeEntry(entry, opts)
	if err != nil {
		return err
	}

	// Check if we are at the root
	if file.dver == nil {
		return nil
	}

	// Update directory pointer by adding file TreeNode to the dir Tree
	tn := block.NewFileTreeNode(filepath.Base(file.Name()), ver.ID)
	file.tree.AddNodes(tn)

	tid, err := file.dev.SetBlock(file.tree)
	if err != nil {
		return err
	}

	if err = file.dver.UpdateVersion(activeVersion, tid); err != nil {
		return err
	}

	edata, err = file.dver.MarshalBinary()
	if err != nil {
		return err
	}

	// Submit directory entry
	entry, opts, err = file.hexlog.NewEntryFrom(file.dver.entry)
	if err != nil {
		return err
	}
	entry.Data = append([]byte{OpFsSet}, edata...)

	opts.WaitBallot = true
	err = file.hexlog.ProposeEntry(entry, opts)
	if err != nil {
		return err
	}

	// TODO:
	// May need to wait for the future here
	// Reload in memory object

	return err
}
