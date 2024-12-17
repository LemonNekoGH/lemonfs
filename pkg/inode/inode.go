package inode

import (
	"context"
	"encoding/json"
	"log"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type LemonFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
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
	Name    string                `json:"name"`
	Content []LemonDirectoryChild `json:"content"`
}

type LemonInode struct {
	fs.Inode

	Content *LemonDirectoryChild
}

// type checks
var _ fs.NodeOnAdder = (*LemonInode)(nil)
var _ fs.NodeReaddirer = (*LemonInode)(nil)

func (i *LemonInode) OnAdd(ctx context.Context) {
	log.Println("Filesystem is mounted")
}

func (i *LemonInode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	if i.Content.Type == "file" {
		return nil, syscall.ENOENT
	}

	entries := []fuse.DirEntry{}
	for _, child := range i.Content.Directory.Content {
		log.Println("Child", child.Type)
		if child.File != nil {
			entries = append(entries, fuse.DirEntry{Name: child.File.Name, Mode: fuse.S_IFREG})
		}
		if child.Directory != nil {
			entries = append(entries, fuse.DirEntry{Name: child.Directory.Name, Mode: fuse.S_IFDIR})
		}
	}

	return fs.NewListDirStream(entries), 0
}
