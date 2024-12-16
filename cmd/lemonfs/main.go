package main

import (
	"context"
	"log"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type myNode struct {
	fs.Inode
}

// Node types must be InodeEmbedders
var _ = (fs.InodeEmbedder)((*myNode)(nil))

// Node types should implement some file system operations, eg. Lookup
var _ = (fs.NodeLookuper)((*myNode)(nil))

func (n *myNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	ops := myNode{}
	out.Mode = 0755
	out.Size = 42
	return n.NewInode(ctx, &ops, fs.StableAttr{Mode: syscall.S_IFREG}), 0
}

func main() {
	server, err := fs.Mount("/tmp/lemonfs", &myNode{}, &fs.Options{})
	if err != nil {
		log.Fatal(err)
	}
	server.Wait()
}
