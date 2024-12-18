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
	"last_accessed_at": 1734525498,
	"last_modified_at": 1734525498,
	"created_at": 1734525498,
	"content": [
		{
			"type": "file",
			"name": "a_file.txt",
			"content": "Hello, World!",
			"last_accessed_at": 1734525498,
			"last_modified_at": 1734525498,
			"created_at": 1734524498
		},
		{
			"type": "directory",
			"name": "a_directory",
			"last_accessed_at": 1734525498,
			"last_modified_at": 1734525498,
			"created_at": 1734523498,
			"content": [
				{
					"type": "file",
					"name": "another_file.txt",
					"content": "Hello, another world!",
					"last_accessed_at": 1734525498,
					"last_modified_at": 1734525498,
					"created_at": 1734525498
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
