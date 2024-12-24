package inode

import (
	"context"
	"log"
	"sync"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/lemonnekogh/lemonfs/pkg/file"
	"github.com/lemonnekogh/lemonfs/pkg/filehandle"
	"github.com/samber/lo"
)

type LemonInode struct {
	fs.Inode

	rwLock sync.RWMutex

	Content *file.LemonDirectoryChild
}

func (i *LemonInode) createFileInode(ctx context.Context, name string) (*fs.Inode, fs.FileHandle) {
	now := uint64(time.Now().Unix())

	newFile := file.LemonDirectoryChild{
		Type: "file",
		File: &file.LemonFile{
			Type:           "file",
			Name:           name,
			Content:        "",
			CreatedAt:      now,
			LastAccessedAt: now,
			LastModifiedAt: now,
		},

		Parent:     i.Content,
		TargetFile: i.Content.TargetFile,
	}

	i.Content.Directory.Content = append(i.Content.Directory.Content, newFile)
	i.Content.WriteToFile()

	lemonInode := &LemonInode{
		Content: &newFile,
	}

	return i.NewInode(ctx, lemonInode, fs.StableAttr{Mode: fuse.S_IFREG}), filehandle.NewLemonFileHandle(&newFile)
}

// type checks
var _ fs.NodeOnAdder = (*LemonInode)(nil)
var _ fs.NodeReaddirer = (*LemonInode)(nil)
var _ fs.NodeLookuper = (*LemonInode)(nil)
var _ fs.NodeGetattrer = (*LemonInode)(nil)
var _ fs.NodeOpener = (*LemonInode)(nil)
var _ fs.NodeCreater = (*LemonInode)(nil)
var _ fs.NodeSetattrer = (*LemonInode)(nil)
var _ fs.NodeRenamer = (*LemonInode)(nil)

func (i *LemonInode) OnAdd(ctx context.Context) {
	log.Println("OnAdd", i.Content.Path())
}

func (i *LemonInode) findChild(name string) (file.LemonDirectoryChild, bool) {
	return lo.Find(i.Content.Directory.Content, func(child file.LemonDirectoryChild) bool {
		if child.File != nil && child.File.Name == name {
			return true
		}

		if child.Directory != nil && child.Directory.Name == name {
			return true
		}

		return false
	})
}

func (i *LemonInode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()

	log.Println("Readdir", i.Content.Path())

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

	log.Printf("Lookup %s in %s", name, i.Content.Path())

	found, ok := i.findChild(name)

	if !ok {
		return nil, syscall.ENOENT
	}

	foundInode := &LemonInode{
		Content: &found,
	}

	mode := uint32(lo.Ternary(found.Type == "file", fuse.S_IFREG, fuse.S_IFDIR))

	return i.NewInode(ctx, foundInode, fs.StableAttr{Mode: mode}), 0
}

func (i *LemonInode) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()

	log.Println("Getattr", i.Content.Path())

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

	log.Printf("Open %s, flags %d", i.Content.Path(), flags)

	if i.Content.File == nil {
		return i, 0, syscall.EISDIR
	}

	if flags&syscall.O_TRUNC != 0 {
		i.Content.File.Content = ""
	}

	return filehandle.NewLemonFileHandle(i.Content), 0, 0
}

func (i *LemonInode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	i.rwLock.Lock()
	defer i.rwLock.Unlock()

	log.Printf("Create %s in %s, flags: %d, mode: %d", name, i.Content.Path(), flags, mode)

	if i.Content.Directory == nil {
		return nil, nil, 0, syscall.ENOTDIR
	}

	// check mode
	if mode&syscall.S_IFMT != syscall.S_IFREG {
		return nil, nil, 0, syscall.ENOTSUP
	}

	// check if the file already exists
	file, ok := i.findChild(name)
	if ok {
		// if must create a new file
		if flags&(syscall.O_CREAT|syscall.O_EXCL) == 0 {
			return nil, nil, 0, syscall.EEXIST
		}

		if flags&syscall.O_CREAT != 0 {
			return nil, nil, 0, syscall.EEXIST
		}

		// create or open an existing file
		return i.NewInode(ctx, &LemonInode{
			Content: &file,
		}, fs.StableAttr{Mode: fuse.S_IFREG}), filehandle.NewLemonFileHandle(&file), 0, 0
	}

	newFile, newFileHandle := i.createFileInode(ctx, name)
	return newFile, newFileHandle, 0, 0
}

func (i *LemonInode) Setattr(ctx context.Context, fh fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	i.rwLock.Lock()
	defer i.rwLock.Unlock()

	log.Printf("Set attr of %s", i.Content.Path())

	return 0
}

func (i *LemonInode) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	i.rwLock.Lock()
	defer i.rwLock.Unlock()
	defer i.Content.WriteToFile()

	if i.Content.Directory == nil {
		return syscall.ENOTDIR
	}

	newChildren := []file.LemonDirectoryChild{}
	ok := false
	var source *file.LemonDirectoryChild
	for _, f := range i.Content.Directory.Content {
		if f.Name() == name {
			source = &f
			ok = true
			continue
		}

		newChildren = append(newChildren, f)
	}

	if !ok {
		return syscall.ENOENT
	}

	targetParent, ok := newParent.(*LemonInode)
	if !ok {
		return syscall.ENOTSUP
	}

	log.Printf("rename %s in %s to %s in %s", name, i.Content.Path(), newName, targetParent.Content.Path())

	// no need to move
	if targetParent.Content.Path() == i.Content.Path() && name == newName {
		return 0
	}

	existsTarget, ok := targetParent.findChild(newName)
	if !ok {
		// move directly
		source.Rename(newName)
		targetParent.Content.Directory.Content = []file.LemonDirectoryChild{*source}
		i.Content.Directory.Content = newChildren
		return 0
	}

	if source.IsFile() {
		if targetParent.Content.IsDirectory() {
			return syscall.EISDIR
		}

		// overwrite the file
		existsTarget.File.Content = source.File.Content
		return 0
	}

	if targetParent.Content.IsFile() {
		return syscall.ENOTDIR
	}

	if len(targetParent.Content.Directory.Content) != 0 {
		return syscall.ENOTEMPTY
	}

	// move
	source.Rename(newName)
	targetParent.Content.Directory.Content = []file.LemonDirectoryChild{*source}
	i.Content.Directory.Content = newChildren

	return 0
}
