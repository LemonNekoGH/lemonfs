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
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/lemonnekogh/lemonfs/pkg/file"
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

	jsonRoot := &file.LemonDirectoryChild{}
	err = json.Unmarshal(jsonContent, jsonRoot)
	if err != nil {
		log.Fatal(err)
	}
	jsonRoot.TargetFile = jsonFile
	jsonRoot.ApplyParentAndTarget(nil)
	if jsonRoot.File == nil && jsonRoot.Directory == nil {
		log.Println("jsonRoot is nil, create empty directory")

		jsonRoot.Directory = &file.LemonDirectory{
			Type:    "directory",
			Content: []file.LemonDirectoryChild{},
		}
		jsonRoot.WriteToFile()
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	rootInode := inode.NewLemonInode(jsonRoot, nil)

	server, err := fs.Mount(mountPoint, rootInode, &fs.Options{
		MountOptions: fuse.MountOptions{
			Debug: true,
		},
	}) // It will call OnAdd
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
	err = server.Unmount()
	if err != nil {
		log.Fatal(err)
	}
}
