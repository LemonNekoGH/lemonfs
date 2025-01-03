package inode_test

import (
	"os"
	"path/filepath"
	"syscall"
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
	r.True(stat.Mode().IsRegular())

	// dirB
	stat, err = os.Stat(filepath.Join(tmpDir, "b"))
	r.NoError(err)
	r.True(stat.Mode().IsDir())

	// not exists
	_, err = os.Stat(filepath.Join(tmpDir, "c"))
	r.Error(err)
	r.True(os.IsNotExist(err))
}

func TestRename(t *testing.T) {
	t.Run("rename file", func(t *testing.T) {
		r := require.New(t)

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
		r.Equal("c", fileA.Name)

		_, err = os.Stat(filepath.Join(tmpDir, "b", "c"))
		r.NoError(err)
	})

	t.Run("rename directory", func(t *testing.T) {
		r := require.New(t)

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
		r.Equal("b", dirA.Name)

		_, err = os.Stat(filepath.Join(tmpDir, "a"))
		r.True(os.IsNotExist(err), "should not exist")

		_, err = os.Stat(filepath.Join(tmpDir, "b"))
		r.NoError(err, "should move successfully")
	})

	t.Run("move cross directory", func(t *testing.T) {
		r := require.New(t)

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
		r.Equal("a", fileA.Name)

		_, err = os.Stat(filepath.Join(tmpDir, "c", "a"))
		r.True(os.IsNotExist(err), "should not exist")

		_, err = os.Stat(filepath.Join(tmpDir, "d", "a"))
		r.NoError(err, "should move successfully")

		_, err = os.Stat(filepath.Join(tmpDir, "d", "b"))
		r.NoError(err, "should not be affected")
	})

	t.Run("rename file to existing file", func(t *testing.T) {
		r := require.New(t)

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
		r.Equal("hello", fileB.Content)

		_, err = os.Stat(filepath.Join(tmpDir, "c", "a"))
		r.True(os.IsNotExist(err), "should not exist")

		_, err = os.Stat(filepath.Join(tmpDir, "c", "b"))
		r.NoError(err, "should move successfully")
	})

	t.Run("not exists", func(t *testing.T) {
		r := require.New(t)

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
		r.True(os.IsNotExist(err))
	})

	t.Run("source is directory but target is file", func(t *testing.T) {
		r := require.New(t)

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
		r := require.New(t)

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

func TestReaddir(t *testing.T) {
	t.Run("is file", func(t *testing.T) {
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

		_, err = os.ReadDir(filepath.Join(tmpDir, "a"))
		r.Error(err)
		r.ErrorIs(err, syscall.ENOTDIR)
	})

	t.Run("is directory", func(t *testing.T) {
		r := require.New(t)

		dir := &file.LemonDirectory{
			Type: "directory",
			Name: "a",
			Content: []file.LemonDirectoryChild{
				{
					Type: "file",
					File: &file.LemonFile{
						Type: "file",
						Name: "b",
					},
				},
				{
					Type: "directory",
					Directory: &file.LemonDirectory{
						Name: "c",
						Type: "directory",
					},
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
						Directory: dir,
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

		entries, err := os.ReadDir(filepath.Join(tmpDir, "a"))
		r.NoError(err)
		r.Equal(2, len(entries))
		r.Equal("b", entries[0].Name())
		r.False(entries[0].IsDir())

		r.Equal("c", entries[1].Name())
		r.True(entries[1].IsDir())
	})
}

func TestMkdir(t *testing.T) {
	t.Run("is file", func(t *testing.T) {
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

		err = os.Mkdir(filepath.Join(tmpDir, "a", "b"), 0755)
		r.ErrorIs(err, syscall.ENOTDIR)
	})

	t.Run("exists", func(t *testing.T) {
		r := require.New(t)

		dirA := &file.LemonDirectory{
			Type: "directory",
			Name: "a",
		}

		root := inode.NewLemonInode(&file.LemonDirectoryChild{
			Type: "directory",
			Directory: &file.LemonDirectory{
				Name: "root",
				Type: "directory",
				Content: []file.LemonDirectoryChild{
					{Type: "directory", Directory: dirA},
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

		err = os.Mkdir(filepath.Join(tmpDir, "a"), 0755)
		r.ErrorIs(err, syscall.EEXIST)
	})

	t.Run("success", func(t *testing.T) {
		r := require.New(t)

		dirA := &file.LemonDirectory{
			Type: "directory",
			Name: "a",
		}

		root := inode.NewLemonInode(&file.LemonDirectoryChild{
			Type: "directory",
			Directory: &file.LemonDirectory{
				Name: "root",
				Type: "directory",
				Content: []file.LemonDirectoryChild{
					{Type: "directory", Directory: dirA},
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

		err = os.Mkdir(filepath.Join(tmpDir, "a", "b"), 0755)
		r.NoError(err)

		dir, err := os.Stat(filepath.Join(tmpDir, "a", "b"))
		r.NoError(err)
		r.True(dir.IsDir())
	})
}

func TestTruncate(t *testing.T) {
	r := require.New(t)

	fileA := &file.LemonFile{
		Type:    "file",
		Name:    "a",
		Content: "hello world",
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

	f, err := os.OpenFile(filepath.Join(tmpDir, "a"), os.O_TRUNC, 0644)
	r.NoError(err)
	defer f.Close()

	r.Equal("", fileA.Content)
}
