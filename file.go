package fidias

import (
	"os"

	"github.com/hexablock/blox/block"
	"github.com/hexablock/blox/filesystem"
	"github.com/hexablock/hexatype"
)

type File struct {
	*filesystem.BloxFile

	versions *VersionedFile
	hexlog   *Hexalog
}

func (file *File) Name() string {
	return file.versions.name
}

// func (file *File) Size() int64 {
// 	return file.fh.Size()
// }

// func (file *File) Sys() interface{} {
// 	return file.versions
// }

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

	// TODO: May need to wait for the future here

	return err
}
