package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type myNode struct {
	fs.Inode
}

// type check
var _ = (fs.NodeOnAdder)((*myNode)(nil))

// This will be called when the file system is mounted
func (n *myNode) OnAdd(ctx context.Context) {
	log.Println("OnAdd")
}

// type check
var _ = (fs.NodeOnForgetter)((*myNode)(nil))

func (n *myNode) OnForget() {
	log.Println("OnForget")
}

// type check
var _ = (fs.NodeLookuper)((*myNode)(nil))

func (n *myNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	log.Println("Lookup", name)
	return nil, syscall.ENOENT
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	server, err := fs.Mount("/tmp/lemonfs", &myNode{}, &fs.Options{}) // It will call OnAdd
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
	err = server.Unmount()
	if err != nil {
		log.Fatal(err)
	}
}
