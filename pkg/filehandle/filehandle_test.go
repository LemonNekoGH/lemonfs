package filehandle_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/lemonnekogh/lemonfs/pkg/file"
	"github.com/lemonnekogh/lemonfs/pkg/inode"
	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	r := require.New(t)

	fileA := &file.LemonFile{
		Type: "file",
		Name: "a",
	}

	root := inode.NewLemonInode(&file.LemonDirectoryChild{
		Type: "directory",
		Directory: &file.LemonDirectory{
			Name: "root",
			Type: "directory",
			Content: []file.LemonDirectoryChild{
				{Type: "file", File: fileA},
			},
		},
	}, nil)

	tmpDir := t.TempDir()
	server, err := fs.Mount(tmpDir, root, &fs.Options{
		MountOptions: fuse.MountOptions{
			Debug: true,
		},
	})
	r.NoError(err)
	defer server.Unmount()

	err = os.WriteFile(filepath.Join(tmpDir, "a"), []byte("hello"), 0644)
	r.NoError(err)
	r.Equal(fileA.Content, "hello")
}

func TestRead(t *testing.T) {
	r := require.New(t)

	fileA := &file.LemonFile{
		Type:    "file",
		Name:    "a",
		Content: "hello",
	}

	root := inode.NewLemonInode(&file.LemonDirectoryChild{
		Type: "directory",
		Directory: &file.LemonDirectory{
			Name: "root",
			Type: "directory",
			Content: []file.LemonDirectoryChild{
				{Type: "file", File: fileA},
			},
		},
	}, nil)

	tmpDir := t.TempDir()
	server, err := fs.Mount(tmpDir, root, &fs.Options{
		MountOptions: fuse.MountOptions{
			Debug: true,
		},
	})
	r.NoError(err)
	defer server.Unmount()

	content, err := os.ReadFile(filepath.Join(tmpDir, "a"))
	r.NoError(err)
	r.Equal(string(content), "hello")
}
