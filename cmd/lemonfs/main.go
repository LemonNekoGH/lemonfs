package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/lemonnekogh/lemonfs/pkg/inode"
)

const jsonContent = `{
	"type": "directory",
	"name": "root",
	"content": [
		{
			"type": "file",
			"name": "a_file.txt",
			"content": "Hello, World!"
		},
		{
			"type": "directory",
			"name": "a_directory",
			"content": [
				{
					"type": "file",
					"name": "another_file.txt",
					"content": "Hello, another world!"
				}
			]
		}
	]
}`

func main() {
	rootNode := &inode.LemonDirectoryChild{}
	err := json.Unmarshal([]byte(jsonContent), rootNode)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	server, err := fs.Mount("/tmp/lemonfs", &inode.LemonInode{Content: rootNode}, &fs.Options{}) // It will call OnAdd
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
	err = server.Unmount()
	if err != nil {
		log.Fatal(err)
	}
}
