package filehandle

import (
	"context"
	"log"
	"sync"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/lemonnekogh/lemonfs/pkg/file"
)

type LemonFileHandle struct {
	file  *file.LemonDirectoryChild
	flags uint32

	rwLock sync.RWMutex
}

func NewLemonFileHandle(file *file.LemonDirectoryChild, flags uint32) *LemonFileHandle {
	return &LemonFileHandle{
		file:  file,
		flags: flags,
	}
}

// type check
var _ fs.FileReader = (*LemonFileHandle)(nil)
var _ fs.FileWriter = (*LemonFileHandle)(nil)
var _ fs.FileSetattrer = (*LemonFileHandle)(nil)

func (fh *LemonFileHandle) Write(ctx context.Context, data []byte, off int64) (uint32, syscall.Errno) {
	fh.rwLock.Lock()
	defer fh.rwLock.Unlock()

	// append mode
	if fh.flags&syscall.O_APPEND != 0 {
		log.Printf("Write %s at %d, %d bytes, append mode", fh.file.Path(), off, len(data))

		fh.file.File.Content += string(data)
		fh.file.WriteToFile()

		return uint32(len(data)), 0
	}

	// normal mode
	log.Printf("Write %s at %d, %d bytes, normal mode", fh.file.Path(), off, len(data))

	fh.file.File.Content = string(data)
	fh.file.WriteToFile()

	return uint32(len(data)), 0
}

func (fh *LemonFileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	fh.rwLock.RLock()
	defer fh.rwLock.RUnlock()

	log.Printf("Read %s at %d, %d bytes, %d bytes available", fh.file.Path(), off, len(dest), len(fh.file.File.Content))

	endIndex := off + int64(len(dest))
	if endIndex > int64(len(fh.file.File.Content)) {
		endIndex = int64(len(fh.file.File.Content))
	}

	readBytes := fh.file.File.Content[off:endIndex]

	return fuse.ReadResultData([]byte(readBytes)), 0
}

func (fh *LemonFileHandle) Setattr(ctx context.Context, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	fh.rwLock.Lock()
	defer fh.rwLock.Unlock()

	log.Printf("Set attr of %s\n", fh.file.Path())

	fh.file.File.CreatedAt = in.Ctime
	fh.file.File.LastModifiedAt = in.Mtime
	fh.file.File.LastAccessedAt = in.Atime

	fh.file.WriteToFile()

	return 0
}
