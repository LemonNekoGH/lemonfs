# lemonfs

A simple FUSE filesystem for learning. It can mount a JSON file as a filesystem.

## Usage

```bash
git clone https://github.com/lemonnekogh/lemonfs.git
cd lemonfs
go run ./cmd/lemonfs/main.go
```

## TODO

- [x] List files in a directory
- [ ] File operations
    - [x] Open
    - [x] Read
    - [ ] Write
- [ ] Directory operations
- [ ] Symlink operations
- [ ] Hardlink operations
