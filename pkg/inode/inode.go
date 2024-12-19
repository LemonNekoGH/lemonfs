package inode

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/samber/lo"
)

type LemonFile struct {
	Type           string `json:"type"`
	Name           string `json:"name"`
	Content        string `json:"content"`
	CreatedAt      int64  `json:"created_at"`
	LastAccessedAt int64  `json:"last_accessed_at"`
	LastModifiedAt int64  `json:"last_modified_at"`
}

type LemonDirectoryChild struct {
	Type      string          `json:"type"`
	File      *LemonFile      `json:"file"`
	Directory *LemonDirectory `json:"directory"`
}

func (c *LemonDirectoryChild) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	if raw["type"] == "file" {
		f := &LemonFile{}
		err = json.Unmarshal(data, f)
		if err != nil {
			return err
		}
		*c = LemonDirectoryChild{Type: "file", File: f}
	} else if raw["type"] == "directory" {
		d := &LemonDirectory{}
		err = json.Unmarshal(data, d)
		if err != nil {
			return err
		}
		*c = LemonDirectoryChild{Type: "directory", Directory: d}
	}

	return nil
}

func (c *LemonDirectoryChild) MarshalJSON() ([]byte, error) {
	if c.File != nil {
		return json.Marshal(c.File)
	}

	if c.Directory != nil {
		return json.Marshal(c.Directory)
	}

	return nil, nil
}

type LemonDirectory struct {
	Type           string                `json:"type"`
	Name           string                `json:"name"`
	Content        []LemonDirectoryChild `json:"content"`
	CreatedAt      int64                 `json:"created_at"`
	LastAccessedAt int64                 `json:"last_accessed_at"`
	LastModifiedAt int64                 `json:"last_modified_at"`
}

type LemonInode struct {
	fs.Inode

	rwLock sync.RWMutex

	Content *LemonDirectoryChild

	Parent     *LemonInode
	TargetFile string
}

func (i *LemonInode) root() *LemonInode {
	root := i
	for root.Parent != nil {
		root = root.Parent
	}

	return root
}

func (i *LemonInode) name() string {
	if i.Content.Type == "file" {
		return i.Content.File.Name
	}

	if i.Content.Type == "directory" {
		return i.Content.Directory.Name
	}

	return ""
}

func (i *LemonInode) Path() string {
	if i.Parent == nil {
		return "/"
	}

	return filepath.Join(i.Parent.Path(), i.name())
}

func (i *LemonInode) WriteToFile() error {
	jsonContent, err := json.Marshal(i.root().Content)
	if err != nil {
		return err
	}

	return os.WriteFile(i.TargetFile, jsonContent, 0644)
}

func (i *LemonInode) createFileInode(ctx context.Context, name string) (*fs.Inode, fs.FileHandle) {
	now := time.Now().Unix()
	newFile := &LemonFile{
		Type:           "file",
		Name:           name,
		Content:        "",
		CreatedAt:      now,
		LastAccessedAt: now,
		LastModifiedAt: now,
	}

	newFileChild := LemonDirectoryChild{Type: "file", File: newFile}

	i.Content.Directory.Content = append(i.Content.Directory.Content, newFileChild)
	i.WriteToFile()

	lemonInode := &LemonInode{
		Content:    &newFileChild,
		Parent:     i,
		TargetFile: i.TargetFile,
	}

	return i.NewInode(ctx, lemonInode, fs.StableAttr{Mode: fuse.S_IFREG}), NewLemonFileHandle(newFile, lemonInode)
}

// type checks
var _ fs.NodeOnAdder = (*LemonInode)(nil)
var _ fs.NodeReaddirer = (*LemonInode)(nil)
var _ fs.NodeLookuper = (*LemonInode)(nil)
var _ fs.NodeGetattrer = (*LemonInode)(nil)
var _ fs.NodeOpener = (*LemonInode)(nil)
var _ fs.NodeCreater = (*LemonInode)(nil)

func (i *LemonInode) OnAdd(ctx context.Context) {
	log.Println("OnAdd", i.Path())
}

func (i *LemonInode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()

	log.Println("Readdir", i.Path())

	if i.Content.Type == "file" {
		return nil, syscall.ENOENT
	}

	entries := []fuse.DirEntry{}
	for _, child := range i.Content.Directory.Content {
		if child.File != nil {
			log.Println("Child", child.Type, child.File.Name)
			entries = append(entries, fuse.DirEntry{Name: child.File.Name, Mode: fuse.S_IFREG})
		}
		if child.Directory != nil {
			log.Println("Child", child.Type, child.Directory.Name)
			entries = append(entries, fuse.DirEntry{Name: child.Directory.Name, Mode: fuse.S_IFDIR})
		}
	}

	return fs.NewListDirStream(entries), 0
}

func (i *LemonInode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()

	log.Printf("Lookup %s in %s", name, i.Path())

	found, ok := lo.Find(i.Content.Directory.Content, func(child LemonDirectoryChild) bool {
		if child.File != nil && child.File.Name == name {
			return true
		}

		if child.Directory != nil && child.Directory.Name == name {
			return true
		}

		return false
	})

	if !ok {
		return nil, syscall.ENOENT
	}

	foundInode := &LemonInode{
		Content:    &found,
		Parent:     i,
		TargetFile: i.TargetFile,
	}

	mode := uint32(lo.Ternary(found.Type == "file", fuse.S_IFREG, fuse.S_IFDIR))

	return i.NewInode(ctx, foundInode, fs.StableAttr{Mode: mode}), 0
}

func (i *LemonInode) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()

	log.Println("Getattr", i.Path())

	if i.Content.Type == "file" {
		out.Size = uint64(len(i.Content.File.Content))
		out.Atime = uint64(i.Content.File.LastAccessedAt)
		out.Mtime = uint64(i.Content.File.LastModifiedAt)
		out.Ctime = uint64(i.Content.File.CreatedAt)
		out.Mode = fuse.S_IFREG
	}

	if i.Content.Type == "directory" {
		out.Atime = uint64(i.Content.Directory.LastAccessedAt)
		out.Mtime = uint64(i.Content.Directory.LastModifiedAt)
		out.Ctime = uint64(i.Content.Directory.CreatedAt)
		out.Mode = fuse.S_IFDIR
	}

	return 0
}

func (i *LemonInode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()

	log.Println("Open", i.Path())

	if i.Content.File != nil {
		return NewLemonFileHandle(i.Content.File, i), 0, 0
	}

	return i, 0, syscall.EISDIR
}

func (i *LemonInode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	i.rwLock.Lock()
	defer i.rwLock.Unlock()

	log.Printf("Create %s in %s, flags: %d, mode: %d", name, i.Path(), flags, mode)

	if i.Content.Directory == nil {
		return nil, nil, 0, syscall.ENOTDIR
	}

	// check mode
	if mode&syscall.S_IFMT != syscall.S_IFREG {
		return nil, nil, 0, syscall.ENOTSUP
	}

	// check if the file already exists
	if file, ok := lo.Find(i.Content.Directory.Content, func(child LemonDirectoryChild) bool {
		return child.File != nil && child.File.Name == name
	}); ok {
		// if must create a new file
		if flags&(syscall.O_CREAT|syscall.O_EXCL) == 0 {
			return nil, nil, 0, syscall.EEXIST
		}

		if flags&syscall.O_CREAT != 0 {
			return nil, nil, 0, syscall.EEXIST
		}

		// create or open an existing file
		return i.NewInode(ctx, &LemonInode{
			Content:    &file,
			Parent:     i,
			TargetFile: i.TargetFile,
		}, fs.StableAttr{Mode: fuse.S_IFREG}), NewLemonFileHandle(file.File, i), 0, 0
	}

	newFile, newFileHandle := i.createFileInode(ctx, name)
	return newFile, newFileHandle, 0, 0
}

// TODO: extract to file_handle.go

type LemonFileHandle struct {
	file *LemonFile

	rwLock sync.RWMutex

	inode *LemonInode
}

func NewLemonFileHandle(file *LemonFile, inode *LemonInode) *LemonFileHandle {
	return &LemonFileHandle{
		file:  file,
		inode: inode,
	}
}

// type check
var _ fs.FileReader = (*LemonFileHandle)(nil)
var _ fs.FileWriter = (*LemonFileHandle)(nil)

func (fh *LemonFileHandle) Write(ctx context.Context, data []byte, off int64) (uint32, syscall.Errno) {
	fh.rwLock.Lock()
	defer fh.rwLock.Unlock()

	log.Printf("Write %s at %d, %d bytes", fh.inode.Path(), off, len(data))

	content := []byte(fh.file.Content)
	// If the offset is greater than the current length, append the missing bytes
	if off > int64(len(content)) {
		content = append(content, make([]byte, off-int64(len(content)))...)
	}

	// Should overwrite the bytes after the offset
	content = append(content[:off], data...)

	fh.file.Content = string(content)

	fh.inode.WriteToFile()

	// Update the last modified time
	fh.file.LastModifiedAt = time.Now().Unix()

	return uint32(len(data)), 0
}

func (fh *LemonFileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	fh.rwLock.RLock()
	defer fh.rwLock.RUnlock()

	log.Printf("Read %s at %d, %d bytes, %d bytes available", fh.inode.Path(), off, len(dest), len(fh.file.Content))

	endIndex := off + int64(len(dest))
	if endIndex > int64(len(fh.file.Content)) {
		endIndex = int64(len(fh.file.Content))
	}

	readBytes := fh.file.Content[off:endIndex]

	return fuse.ReadResultData([]byte(readBytes)), 0
}
