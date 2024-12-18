package inode

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"syscall"

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

	mu sync.Mutex

	Content *LemonDirectoryChild
}

// type checks
var _ fs.NodeOnAdder = (*LemonInode)(nil)
var _ fs.NodeReaddirer = (*LemonInode)(nil)
var _ fs.NodeLookuper = (*LemonInode)(nil)
var _ fs.NodeGetattrer = (*LemonInode)(nil)

func (i *LemonInode) OnAdd(ctx context.Context) {
	log.Println("Filesystem is mounted")
}

func (i *LemonInode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	i.mu.Lock()
	defer i.mu.Unlock()

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
	i.mu.Lock()
	defer i.mu.Unlock()

	log.Println("Lookup", name, i.Content.Type)

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

	if found.Type == "file" {
		return i.NewInode(ctx, &LemonInode{Content: &found}, fs.StableAttr{Mode: fuse.S_IFREG}), 0
	}

	if found.Type == "directory" {
		return i.NewInode(ctx, &LemonInode{Content: &found}, fs.StableAttr{Mode: fuse.S_IFDIR}), 0
	}

	return nil, syscall.ENOENT
}

func (i *LemonInode) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	i.mu.Lock()
	defer i.mu.Unlock()

	log.Println("Getattr", i.Content.Type)

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
