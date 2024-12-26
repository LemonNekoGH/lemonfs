package file

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type LemonFile struct {
	Type           string `json:"type"`
	Name           string `json:"name"`
	Content        string `json:"content"`
	CreatedAt      uint64 `json:"created_at"`
	LastAccessedAt uint64 `json:"last_accessed_at"`
	LastModifiedAt uint64 `json:"last_modified_at"`
}

type LemonDirectoryChild struct {
	Type      string          `json:"type"`
	File      *LemonFile      `json:"file"`
	Directory *LemonDirectory `json:"directory"`

	Parent     *LemonDirectoryChild
	TargetFile string
}

func (c *LemonDirectoryChild) IsFile() bool {
	return c.File != nil && c.Directory == nil
}

func (c *LemonDirectoryChild) IsDirectory() bool {
	return c.File == nil && c.Directory != nil
}

func (c *LemonDirectoryChild) Rename(newName string) {
	if c.IsFile() {
		c.File.Name = newName
		return
	}

	c.Directory.Name = newName
}

func (c *LemonDirectoryChild) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	if raw["type"] == "file" {
		f := &LemonFile{}
		err = json.Unmarshal(data, f)
		if err != nil {
			return err
		}
		*c = LemonDirectoryChild{Type: "file", File: f}
	} else if raw["type"] == "directory" {
		d := &LemonDirectory{}
		err = json.Unmarshal(data, d)
		if err != nil {
			return err
		}
		*c = LemonDirectoryChild{Type: "directory", Directory: d}
	}

	return nil
}

func (c *LemonDirectoryChild) MarshalJSON() ([]byte, error) {
	if c.File != nil {
		return json.Marshal(c.File)
	}

	if c.Directory != nil {
		return json.Marshal(c.Directory)
	}

	return nil, nil
}

func (c *LemonDirectoryChild) WriteToFile() error {
	jsonContent, err := json.Marshal(c.root())
	if err != nil {
		return err
	}

	return os.WriteFile(c.TargetFile, jsonContent, 0644)
}

func (c *LemonDirectoryChild) root() *LemonDirectoryChild {
	root := c
	for root.Parent != nil {
		root = root.Parent
	}

	return root
}

func (c *LemonDirectoryChild) Name() string {
	switch c.Type {
	case "file":
		return c.File.Name
	case "directory":
		return c.Directory.Name
	default:
		return ""
	}
}

func (c *LemonDirectoryChild) Path() string {
	if c.Parent == nil {
		return "/"
	}

	return filepath.Join(c.Parent.Path(), c.Name())
}

func (c *LemonDirectoryChild) ApplyParentAndTarget(parent *LemonDirectoryChild) {
	if parent != nil {
		c.Parent = parent
		c.TargetFile = parent.TargetFile
	}

	if c.Directory != nil {
		for i := range c.Directory.Content {
			c.Directory.Content[i].ApplyParentAndTarget(c)
		}
	}
}

type LemonDirectory struct {
	Type           string                `json:"type"`
	Name           string                `json:"name"`
	Content        []LemonDirectoryChild `json:"content"`
	CreatedAt      uint64                `json:"created_at"`
	LastAccessedAt uint64                `json:"last_accessed_at"`
	LastModifiedAt uint64                `json:"last_modified_at"`
}
