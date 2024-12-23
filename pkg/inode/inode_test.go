package inode_test

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

func TestMain(m *testing.M) {
	m.Run()
}

func TestLookup(t *testing.T) {
	r := require.New(t)

	fileA := &file.LemonFile{
		Type: "file",
		Name: "a",
	}

	dirB := &file.LemonDirectory{
		Type: "directory",
		Name: "b",
		Content: []file.LemonDirectoryChild{
			{
				Type: "file",
				File: fileA,
			},
		},
	}

	root := &inode.LemonInode{
		Content: &file.LemonDirectoryChild{
			Type: "directory",
			Directory: &file.LemonDirectory{
				Name: "root",
				Type: "directory",
				Content: []file.LemonDirectoryChild{
					{
						Type:      "directory",
						Directory: dirB,
					},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	server, err := fs.Mount(tmpDir, root, &fs.Options{
		MountOptions: fuse.MountOptions{
			Debug: true,
		},
	})
	r.NoError(err)
	defer server.Unmount()

	// fileA
	stat, err := os.Stat(filepath.Join(tmpDir, "b", "a"))
	r.NoError(err)
	r.Equal(stat.Mode().IsRegular(), true)

	// dirB
	stat, err = os.Stat(filepath.Join(tmpDir, "b"))
	r.NoError(err)
	r.Equal(stat.Mode().IsDir(), true)

	// not exists
	_, err = os.Stat(filepath.Join(tmpDir, "c"))
	r.Error(err)
	r.Equal(os.IsNotExist(err), true)
}
