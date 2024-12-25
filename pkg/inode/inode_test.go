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

	root := inode.NewLemonInode(&file.LemonDirectoryChild{
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
	}, nil)
	root.Content.ApplyParentAndTarget(nil)

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

func TestRename(t *testing.T) {
	r := require.New(t)

	t.Run("rename file", func(t *testing.T) {
		// path: /b/a
		fileA := &file.LemonFile{
			Type: "file",
			Name: "a",
		}

		// path: /b
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

		root := inode.NewLemonInode(&file.LemonDirectoryChild{
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
		}, nil)

		tmpDir := t.TempDir()
		server, err := fs.Mount(tmpDir, root, &fs.Options{
			MountOptions: fuse.MountOptions{
				Debug: true,
			},
		})
		r.NoError(err)
		defer server.Unmount()

		// rename file
		err = os.Rename(filepath.Join(tmpDir, "b", "a"), filepath.Join(tmpDir, "b", "c"))
		r.NoError(err)
		r.Equal(fileA.Name, "c")

		_, err = os.Stat(filepath.Join(tmpDir, "b", "c"))
		r.NoError(err)
	})

	t.Run("rename directory", func(t *testing.T) {
		dirA := &file.LemonDirectory{
			Type:    "directory",
			Name:    "a",
			Content: []file.LemonDirectoryChild{},
		}

		root := inode.NewLemonInode(&file.LemonDirectoryChild{
			Type: "directory",
			Directory: &file.LemonDirectory{
				Name: "root",
				Type: "directory",
				Content: []file.LemonDirectoryChild{
					{
						Type:      "directory",
						Directory: dirA,
					},
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

		err = os.Rename(filepath.Join(tmpDir, "a"), filepath.Join(tmpDir, "b"))
		r.NoError(err)
		r.Equal(dirA.Name, "b")

		_, err = os.Stat(filepath.Join(tmpDir, "a"))
		r.True(os.IsNotExist(err), "should not exist")

		_, err = os.Stat(filepath.Join(tmpDir, "b"))
		r.NoError(err, "should move successfully")
	})

	t.Run("move cross directory", func(t *testing.T) {
		// path: /c/a
		fileA := &file.LemonFile{
			Type: "file",
			Name: "a",
		}

		// path: /d/b
		fileB := &file.LemonFile{
			Type: "file",
			Name: "b",
		}

		// path: /c
		dirC := &file.LemonDirectory{
			Type: "directory",
			Name: "c",
			Content: []file.LemonDirectoryChild{
				{
					Type: "file",
					File: fileA,
				},
			},
		}

		// path: /d
		dirD := &file.LemonDirectory{
			Type: "directory",
			Name: "d",
			Content: []file.LemonDirectoryChild{
				{
					Type: "file",
					File: fileB,
				},
			},
		}

		root := inode.NewLemonInode(&file.LemonDirectoryChild{
			Type: "directory",
			Directory: &file.LemonDirectory{
				Name: "root",
				Type: "directory",
				Content: []file.LemonDirectoryChild{
					{
						Type:      "directory",
						Directory: dirC,
					},
					{
						Type:      "directory",
						Directory: dirD,
					},
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

		err = os.Rename(filepath.Join(tmpDir, "c", "a"), filepath.Join(tmpDir, "d", "a"))
		r.NoError(err)
		r.Equal(fileA.Name, "a")

		_, err = os.Stat(filepath.Join(tmpDir, "c", "a"))
		r.True(os.IsNotExist(err), "should not exist")

		_, err = os.Stat(filepath.Join(tmpDir, "d", "a"))
		r.NoError(err, "should move successfully")

		_, err = os.Stat(filepath.Join(tmpDir, "d", "b"))
		r.NoError(err, "should not be affected")
	})

	t.Run("rename file to existing file", func(t *testing.T) {
		// path: /c/a
		fileA := &file.LemonFile{
			Type:    "file",
			Name:    "a",
			Content: "hello",
		}

		// path: /c/b
		fileB := &file.LemonFile{
			Type:    "file",
			Name:    "b",
			Content: "world",
		}

		dirC := &file.LemonDirectory{
			Type: "directory",
			Name: "c",
			Content: []file.LemonDirectoryChild{
				{
					Type: "file",
					File: fileA,
				},
				{
					Type: "file",
					File: fileB,
				},
			},
		}

		root := inode.NewLemonInode(&file.LemonDirectoryChild{
			Type: "directory",
			Directory: &file.LemonDirectory{
				Name: "root",
				Type: "directory",
				Content: []file.LemonDirectoryChild{
					{
						Type:      "directory",
						Directory: dirC,
					},
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

		err = os.Rename(filepath.Join(tmpDir, "c", "a"), filepath.Join(tmpDir, "c", "b"))
		r.NoError(err)
		r.Equal(fileB.Content, "hello")

		_, err = os.Stat(filepath.Join(tmpDir, "c", "a"))
		r.True(os.IsNotExist(err), "should not exist")

		_, err = os.Stat(filepath.Join(tmpDir, "c", "b"))
		r.NoError(err, "should move successfully")
	})

	t.Run("not exists", func(t *testing.T) {
		root := inode.NewLemonInode(&file.LemonDirectoryChild{
			Type: "directory",
			Directory: &file.LemonDirectory{
				Name:    "root",
				Type:    "directory",
				Content: []file.LemonDirectoryChild{},
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

		err = os.Rename(filepath.Join(tmpDir, "a"), filepath.Join(tmpDir, "b"))
		r.NoError(err)
		r.Equal(os.IsNotExist(err), true)
	})

	t.Run("source is directory but target is file", func(t *testing.T) {
		fileA := &file.LemonFile{
			Type: "file",
			Name: "a",
		}

		dirB := &file.LemonDirectory{
			Type:    "directory",
			Name:    "b",
			Content: []file.LemonDirectoryChild{},
		}

		root := inode.NewLemonInode(&file.LemonDirectoryChild{
			Type: "directory",
			Directory: &file.LemonDirectory{
				Name: "root",
				Type: "directory",
				Content: []file.LemonDirectoryChild{
					{
						Type: "file",
						File: fileA,
					},
					{
						Type:      "directory",
						Directory: dirB,
					},
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

		err = os.Rename(filepath.Join(tmpDir, "a"), filepath.Join(tmpDir, "b"))
		r.True(os.IsExist(err))
	})

	t.Run("source is file but target is directory", func(t *testing.T) {
		fileA := &file.LemonFile{
			Type: "file",
			Name: "a",
		}

		dirB := &file.LemonDirectory{
			Type:    "directory",
			Name:    "b",
			Content: []file.LemonDirectoryChild{},
		}

		root := inode.NewLemonInode(&file.LemonDirectoryChild{
			Type: "directory",
			Directory: &file.LemonDirectory{
				Name: "root",
				Type: "directory",
				Content: []file.LemonDirectoryChild{
					{
						Type: "file",
						File: fileA,
					},
					{
						Type:      "directory",
						Directory: dirB,
					},
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

		err = os.Rename(filepath.Join(tmpDir, "a"), filepath.Join(tmpDir, "b"))
		r.True(os.IsExist(err))
	})
}
