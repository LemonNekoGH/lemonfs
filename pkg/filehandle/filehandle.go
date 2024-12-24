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
	file *file.LemonDirectoryChild

	rwLock sync.RWMutex
}

func NewLemonFileHandle(file *file.LemonDirectoryChild) *LemonFileHandle {
	return &LemonFileHandle{
		file: file,
	}
}

// type check
var _ fs.FileReader = (*LemonFileHandle)(nil)
var _ fs.FileWriter = (*LemonFileHandle)(nil)
var _ fs.FileSetattrer = (*LemonFileHandle)(nil)

func (fh *LemonFileHandle) Write(ctx context.Context, data []byte, off int64) (uint32, syscall.Errno) {
	fh.rwLock.Lock()
	defer fh.rwLock.Unlock()

	log.Printf("Write %s at %d, %d bytes", fh.file.Path(), off, len(data))

	content := []byte(fh.file.File.Content)
	// If the offset is greater than the current length, append the missing bytes
	if off > int64(len(content)) {
		content = append(content, make([]byte, off-int64(len(content)))...)
	}

	// Should overwrite the bytes after the offset
	content = append(content[:off], data...)

	fh.file.File.Content = string(content)

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
