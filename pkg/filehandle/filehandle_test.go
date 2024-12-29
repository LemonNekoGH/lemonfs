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
	t.Run("normal mode", func(t *testing.T) {
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
		r.Equal("hello", fileA.Content)
	})

	t.Run("append mode", func(t *testing.T) {
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

		f, err := os.OpenFile(filepath.Join(tmpDir, "a"), os.O_APPEND|os.O_WRONLY, 0644)
		r.NoError(err)
		defer f.Close()

		_, err = f.Write([]byte(" world"))
		r.NoError(err)
		r.Equal("hello world", fileA.Content)
	})
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

	r.Equal("hello", fileA.Content)
}
