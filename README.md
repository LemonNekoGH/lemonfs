# lemonfs

A simple FUSE filesystem for learning. It can mount a JSON file as a filesystem.

## System requirements

- Go 1.23+
- FUSE

  ```bash
  sudo apt-get install fuse
  # or
  brew install macfuse # You need to active the system extension
  ```

## Usage

```bash
git clone https://github.com/lemonnekogh/lemonfs.git
cd lemonfs
go run ./cmd/lemonfs/main.go
```

### Errors and solutions

- Transport endpoint is not connected

  ```bash
  fusermount -u <mount_point>
  ```

## TODO

### File

- [x] open
- [x] read
- [x] write
- [ ] close
- [ ] truncate
- [ ] unlink
- [x] stat
- [ ] chmod
- [ ] chown

### Directory

- [x] mkdir
- [ ] rmdir
- [x] readdir
- [x] rename
- [ ] opendir
- [x] lookup

### Link

- [ ] link
- [ ] symlink
- [ ] readlink
- [ ] unlink

### Metadata

- [x] getattr
- [ ] setattr
- [ ] statfs

### Advanced features

- [ ] fsync
- [ ] flush
- [ ] lock
- [ ] access
