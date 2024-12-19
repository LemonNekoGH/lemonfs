package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/lemonnekogh/lemonfs/pkg/inode"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: lemonfs <json_file> <mount_point>")
		os.Exit(1)
	}

	jsonFile := os.Args[1]
	mountPoint := os.Args[2]

	jsonContent, err := os.ReadFile(jsonFile)
	if err != nil {
		log.Fatal(err)
	}

	jsonRoot := &inode.LemonDirectory{}
	err = json.Unmarshal(jsonContent, jsonRoot)
	if err != nil {
		log.Fatal(err)
	}
	jsonRoot.Type = "directory"

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	rootInode := &inode.LemonInode{
		Content: &inode.LemonDirectoryChild{
			Directory: jsonRoot,
		},
		TargetFile: jsonFile,
	}

	server, err := fs.Mount(mountPoint, rootInode, &fs.Options{}) // It will call OnAdd
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
	err = server.Unmount()
	if err != nil {
		log.Fatal(err)
	}
}
