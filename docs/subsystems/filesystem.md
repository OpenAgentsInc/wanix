# Filesystem Subsystem

The filesystem subsystem provides the foundation for Wanix's "everything is a file" architecture. It includes multiple filesystem implementations, each serving specific purposes.

## Overview

The filesystem subsystem consists of:
- **Core abstractions** (`fs/` package)
- **Filesystem toolkit** (`fs/fskit/` package)
- **Protocol implementations** (`fs/p9kit/`, `fs/fusekit/`)
- **Specialized filesystems** (`fs/tarfs/`)

## Core Filesystem Abstractions

### Base Interfaces (`fs/file.go`)

The fundamental file operations all filesystems must support:

```go
// Basic file interface
type File interface {
    io.ReadWriteCloser
    io.Seeker
    
    Readdir(count int) ([]FileInfo, error)
    Stat() (FileInfo, error)
    Sync() error
    Truncate(size int64) error
}

// Filesystem interface
type FS interface {
    Open(name string) (File, error)
}

// Extended filesystem operations
type FSExtra interface {
    FS
    Create(name string) (File, error)
    Mkdir(name string, perm FileMode) error
    Remove(name string) error
    Rename(oldname, newname string) error
}
```

### Path Operations (`fs/internal.go`)

Internal utilities for path manipulation:
- Path cleaning and normalization
- Parent directory resolution
- Path joining with proper separators
- Relative path handling

## Filesystem Toolkit (`fskit`)

The fskit package provides composable building blocks for creating filesystems.

### Memory Filesystem (`fskit/memfs.go`)

A complete in-memory filesystem implementation:

```go
// Create a new memory filesystem
mfs := fskit.NewMemFS()

// Use like any filesystem
f, _ := mfs.Create("/hello.txt")
f.Write([]byte("Hello, World!"))
f.Close()

// Read back
data, _ := fs.ReadFile(mfs, "/hello.txt")
```

Features:
- Full directory hierarchy
- File permissions and timestamps
- Concurrent access safe
- Efficient memory usage

### Map Filesystem (`fskit/mapfs.go`)

A filesystem backed by a simple map, ideal for static content:

```go
// Create filesystem from map
mfs := fskit.MapFS{
    "config.json": &fskit.File{
        Data: []byte(`{"version": "1.0"}`),
    },
    "static/index.html": &fskit.File{
        Data: []byte("<h1>Welcome</h1>"),
    },
}
```

Use cases:
- Embedded assets
- Configuration files
- Static resources
- Virtual files

### Union Filesystem (`fskit/unionfs.go`)

Merges multiple filesystems into a single view:

```go
// Layer multiple filesystems
base := fskit.NewMemFS()
overlay := fskit.NewMemFS()

// Create union
ufs := fskit.NewUnionFS(overlay, base)

// Writes go to overlay, reads check both
f, _ := ufs.Create("/data.txt")  // Creates in overlay
```

Features:
- Read-through to lower layers
- Write to topmost layer
- Directory merging
- Whiteout support for deletions

### Function Files (`fskit/funcfile.go`)

Files that execute functions on read/write:

```go
// Create a dynamic file
timeFile := fskit.NewFuncFile(func() ([]byte, error) {
    return []byte(time.Now().String()), nil
})

// Each read returns current time
mfs.Add("/proc/time", timeFile)
```

Use cases:
- `/proc`-style synthetic files
- Dynamic status information
- Control files
- Device files

### Directory Files (`fskit/dirfile.go`)

Specialized handling for directory operations:

```go
// Create a directory that lists dynamic content
dynDir := fskit.NewDirFile(func() ([]fs.DirEntry, error) {
    // Return current entries
    return getCurrentEntries(), nil
})
```

## TAR Filesystem (`tarfs`)

Read-only filesystem backed by TAR archives:

```go
// Open TAR archive as filesystem
data, _ := os.ReadFile("archive.tar")
tfs, _ := tarfs.New(bytes.NewReader(data))

// Access files directly
f, _ := tfs.Open("path/in/tar/file.txt")
```

Features:
- Lazy extraction
- Memory efficient
- Supports all TAR formats
- Nested archive support

## Plan 9 Protocol (`p9kit`)

Implements the 9P protocol for network filesystems:

### Server Implementation

```go
// Create 9P server from any filesystem
srv := p9kit.NewServer(myFS)

// Serve over network
listener, _ := net.Listen("tcp", ":9999")
srv.Serve(listener)
```

### Client Implementation

```go
// Connect to 9P server
client, _ := p9kit.Dial("tcp", "server:9999")

// Mount as filesystem
fs := client.AsFS()

// Use normally
data, _ := fs.ReadFile(fs, "/remote/file")
```

Features:
- Full 9P2000 protocol support
- Authentication support
- Efficient bulk transfers
- Connection pooling

## FUSE Integration (`fusekit`)

Native filesystem mounting on Mac/Linux:

```go
// Mount any Wanix filesystem natively
mount := fusekit.NewMount(myFS)

// Mount at path
err := mount.Mount("/mnt/wanix")

// Now accessible from any program
// $ ls /mnt/wanix
```

Features:
- Transparent integration
- Full POSIX semantics
- Performance optimizations
- Graceful error handling

## Advanced Patterns

### Filesystem Composition

Build complex filesystems from simple parts:

```go
// Start with memory fs
base := fskit.NewMemFS()

// Add static files
static := fskit.MapFS{
    "version": &fskit.File{Data: []byte("1.0")},
}

// Combine with union
combined := fskit.NewUnionFS(base, static)

// Add synthetic files
combined.Add("/proc/uptime", fskit.NewFuncFile(getUptime))

// Expose over network
srv := p9kit.NewServer(combined)
```

### Custom Filesystem Implementation

Create specialized filesystems:

```go
type GitFS struct {
    repo *git.Repository
}

func (g *GitFS) Open(name string) (fs.File, error) {
    // Resolve path to git object
    obj := g.resolveObject(name)
    
    // Return file interface
    return &gitFile{obj: obj}, nil
}
```

### Performance Optimization

Tips for efficient filesystem operations:

1. **Cache frequently accessed files**
```go
type CachedFS struct {
    fs.FS
    cache map[string][]byte
}
```

2. **Use readahead for sequential access**
```go
func (f *File) Read(p []byte) (int, error) {
    if f.sequential {
        f.readahead()
    }
    return f.read(p)
}
```

3. **Implement stat caching**
```go
type StatCache struct {
    fs    fs.FS
    cache map[string]*cachedStat
}
```

## Security Considerations

### Path Traversal Protection

All filesystem implementations prevent escaping root:

```go
func cleanPath(p string) string {
    // Remove .. components that escape root
    cleaned := path.Clean(p)
    if !strings.HasPrefix(cleaned, "/") {
        cleaned = "/" + cleaned
    }
    return cleaned
}
```

### Permission Enforcement

Filesystems respect Unix permissions:

```go
func (f *file) Write(p []byte) (int, error) {
    if !f.mode.IsWrite() {
        return 0, fs.ErrPermission
    }
    return f.write(p)
}
```

### Resource Limits

Prevent resource exhaustion:

```go
type LimitedFS struct {
    fs.FS
    maxSize  int64
    maxFiles int
}
```

## Best Practices

1. **Use the right filesystem for the job**
   - MemFS for temporary data
   - MapFS for static content
   - UnionFS for layering
   - TarFS for archives

2. **Follow Go conventions**
   - Implement io.Reader/Writer when possible
   - Return appropriate errors (fs.ErrNotExist, etc.)
   - Support context cancellation

3. **Design for composability**
   - Small, focused implementations
   - Standard interfaces
   - Clear semantics

4. **Consider performance**
   - Minimize allocations
   - Cache when appropriate
   - Use buffering for I/O

## Future Enhancements

- **Encryption layer**: Transparent file encryption
- **Compression**: On-the-fly compression/decompression
- **Versioning**: Git-like file history
- **Replication**: Multi-node filesystem sync
- **Quotas**: Storage limit enforcement