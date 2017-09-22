package fidias

import (
	"os"
	"time"

	"github.com/hexablock/blox/filesystem"
)

type File struct {
	versions *VersionedFile
	fh       *filesystem.BloxFile
}

func (file *File) IsDir() bool {
	return file.fh.IsDir()
}

func (file *File) ModTime() time.Time {
	return file.fh.ModTime()
}

func (file *File) Mode() os.FileMode {
	return file.fh.Mode()
}

func (file *File) Name() string {
	return string(file.versions.key)
}

func (file *File) Size() int64 {
	return file.fh.Size()
}

func (file *File) Sys() interface{} {
	return file.versions
}
