package fidias

import (
	"os"

	"github.com/hexablock/blox/block"
	"github.com/hexablock/blox/filesystem"
	"github.com/hexablock/hexatype"
)

// File is a fidias file representing a standard OS file type interface.
// It contains methods to perform native go file type operations
type File struct {
	*filesystem.BloxFile

	versions *VersionedFile
	hexlog   *Hexalog
}

// Name returns the absolute path name of the file
func (file *File) Name() string {
	return file.versions.name
}

// Close closes the underlying BloxFile and updates hexalog with the new
// hash entries
func (file *File) Close() error {
	err := file.BloxFile.Close()
	if err != nil {
		return err
	}

	// Nothing to do if not writing
	if file.Flags() != os.O_WRONLY {
		return nil
	}

	// Update if we are writing
	idx := file.BloxFile.Sys().(*block.IndexBlock)
	ver := file.versions.Version()
	ver.ID = idx.ID()
	if err = file.versions.UpdateVersion(ver); err != nil {
		return err
	}

	edata, err := file.versions.MarshalBinary()
	if err != nil {
		return err
	}

	var (
		entry *hexatype.Entry
		opts  *hexatype.RequestOptions
	)
	entry, opts, err = file.hexlog.NewEntryFrom(file.versions.entry)
	if err != nil {
		return err
	}
	entry.Data = append([]byte{OpFsSet}, edata...)
	ballot, err := file.hexlog.ProposeEntry(entry, opts)
	if err != nil {
		return err
	}
	err = ballot.Wait()

	// TODO:
	// May need to wait for the future here
	// Reload in memory object

	return err
}
