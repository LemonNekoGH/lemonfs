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

func (i *LemonInode) createFileInode(ctx context.Context, name string, flags uint32) (*fs.Inode, fs.FileHandle) {
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

	lemonInode := NewLemonInode(&newFile, i.Content)

	return i.NewInode(ctx, lemonInode, fs.StableAttr{Mode: fuse.S_IFREG}), filehandle.NewLemonFileHandle(&newFile, flags)
}

func (i *LemonInode) createDirectoryInode(ctx context.Context, name string) *fs.Inode {
	now := uint64(time.Now().Unix())
	newDir := file.LemonDirectoryChild{
		Type: "directory",
		Directory: &file.LemonDirectory{
			Type:           "directory",
			Name:           name,
			Content:        []file.LemonDirectoryChild{},
			LastAccessedAt: now,
			LastModifiedAt: now,
			CreatedAt:      now,
		},

		Parent:     i.Content,
		TargetFile: i.Content.TargetFile,
	}

	i.Content.Directory.Content = append(i.Content.Directory.Content, newDir)
	i.Content.WriteToFile()

	lemonInode := NewLemonInode(&newDir, i.Content)

	return i.NewInode(ctx, lemonInode, fs.StableAttr{Mode: fuse.S_IFDIR})
}

func NewLemonInode(content *file.LemonDirectoryChild, parent *file.LemonDirectoryChild) *LemonInode {
	lemonInode := &LemonInode{
		Content: content,
	}

	lemonInode.Content.ApplyParentAndTarget(parent)

	return lemonInode
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
var _ fs.NodeMkdirer = (*LemonInode)(nil)

func (i *LemonInode) OnAdd(ctx context.Context) {
	log.Println("OnAdd", i.Content.Path())
}

func (i *LemonInode) findChild(name string) (file.LemonDirectoryChild, bool) {
	return lo.Find(i.Content.Directory.Content, func(child file.LemonDirectoryChild) bool {
		if child.IsFile() && child.File.Name == name {
			return true
		}

		if child.IsDirectory() && child.Directory.Name == name {
			return true
		}

		return false
	})
}

func (i *LemonInode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()

	log.Println("Readdir", i.Content.Path())

	if i.Content.IsFile() {
		return nil, syscall.ENOTDIR
	}

	entries := []fuse.DirEntry{}
	for _, child := range i.Content.Directory.Content {
		mode := lo.Ternary(child.IsFile(), fuse.S_IFREG, fuse.S_IFDIR)
		entries = append(entries, fuse.DirEntry{Name: child.Name(), Mode: uint32(mode)})
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

	foundInode := NewLemonInode(&found, i.Content)

	mode := uint32(lo.Ternary(found.IsFile(), fuse.S_IFREG, fuse.S_IFDIR))

	return i.NewInode(ctx, foundInode, fs.StableAttr{Mode: mode}), 0
}

func (i *LemonInode) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	i.rwLock.RLock()
	defer i.rwLock.RUnlock()

	log.Println("Getattr", i.Content.Path())

	if i.Content.IsFile() {
		out.Size = uint64(len(i.Content.File.Content))
		out.Atime = uint64(i.Content.File.LastAccessedAt)
		out.Mtime = uint64(i.Content.File.LastModifiedAt)
		out.Ctime = uint64(i.Content.File.CreatedAt)
		out.Mode = fuse.S_IFREG
	}

	if i.Content.IsDirectory() {
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

	log.Printf("Open %s, flags %d, truncate: %t", i.Content.Path(), flags, flags&syscall.O_TRUNC == syscall.O_TRUNC)

	if i.Content.IsDirectory() {
		return i, 0, syscall.EISDIR
	}

	if flags&syscall.O_TRUNC == syscall.O_TRUNC {
		i.Content.File.Content = ""
		i.Content.WriteToFile()
	}

	return filehandle.NewLemonFileHandle(i.Content, flags), 0, 0
}

func (i *LemonInode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	i.rwLock.Lock()
	defer i.rwLock.Unlock()

	log.Printf("Create %s in %s, flags: %d, mode: %d", name, i.Content.Path(), flags, mode)

	if i.Content.IsFile() {
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

		if flags&syscall.O_CREAT == syscall.O_CREAT {
			return nil, nil, 0, syscall.EEXIST
		}

		// create or open an existing file

		foundInode := NewLemonInode(&file, i.Content)
		return i.NewInode(ctx, foundInode, fs.StableAttr{Mode: fuse.S_IFREG}), filehandle.NewLemonFileHandle(&file, flags), 0, 0
	}

	newFile, newFileHandle := i.createFileInode(ctx, name, flags)
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

	if i.Content.IsFile() {
		return syscall.ENOTDIR
	}

	// find and copy the children except the source
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

		if targetParent.Content.Path() == i.Content.Path() {
			i.Content.WriteToFile()
			return 0
		}

		targetParent.Content.Directory.Content = append(targetParent.Content.Directory.Content, *source)
		i.Content.Directory.Content = newChildren

		i.Content.WriteToFile()

		return 0
	}

	if source.IsFile() {
		if existsTarget.IsDirectory() {
			return syscall.EEXIST
		}

		// overwrite the file
		existsTarget.File.Content = source.File.Content
		i.Content.Directory.Content = newChildren

		i.Content.WriteToFile()

		return 0
	}

	if targetParent.Content.IsFile() {
		return syscall.EEXIST
	}

	if len(targetParent.Content.Directory.Content) != 0 {
		return syscall.ENOTEMPTY
	}

	// move
	source.Rename(newName)
	targetParent.Content.Directory.Content = append(targetParent.Content.Directory.Content, *source)
	i.Content.Directory.Content = newChildren

	i.Content.WriteToFile()
	return 0
}

func (i *LemonInode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	i.rwLock.Lock()
	defer i.rwLock.Unlock()

	log.Printf("Mkdir %s in %s", name, i.Content.Path())

	if i.Content.IsFile() {
		return nil, syscall.ENOTDIR
	}

	if _, ok := i.findChild(name); ok {
		return nil, syscall.EEXIST
	}

	return i.createDirectoryInode(ctx, name), 0
}
