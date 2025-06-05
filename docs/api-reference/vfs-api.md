# VFS API Reference

The Virtual Filesystem (VFS) API is the core interface for all file operations in Wanix.

## Core Interfaces

### File Interface

```go
type File interface {
    io.Reader                          // Read([]byte) (int, error)
    io.Writer                          // Write([]byte) (int, error)
    io.Closer                          // Close() error
    io.Seeker                          // Seek(int64, int) (int64, error)
    
    Readdir(count int) ([]FileInfo, error)
    Stat() (FileInfo, error)
    Sync() error
    Truncate(size int64) error
}
```

### FileInfo Interface

```go
type FileInfo interface {
    Name() string       // Base name of the file
    Size() int64       // Length in bytes
    Mode() FileMode    // File mode bits
    ModTime() time.Time // Modification time
    IsDir() bool       // Abbreviation for Mode().IsDir()
    Sys() interface{}  // Underlying data source
}
```

### FS Interface

```go
type FS interface {
    Open(name string) (File, error)
}

type FSExtra interface {
    FS
    Create(name string) (File, error)
    Mkdir(name string, perm FileMode) error
    Remove(name string) error
    Rename(oldname, newname string) error
    Stat(name string) (FileInfo, error)
}
```

## Namespace Operations

### Namespace Type

```go
type Namespace struct {
    // Private fields
}

// Create new namespace
func NewNamespace() *Namespace

// Fork creates a copy-on-write clone
func (ns *Namespace) Fork() *Namespace
```

### Mount Operations

```go
// Mount a filesystem at path
func (ns *Namespace) Mount(fs FS, path string) error

// Unmount a filesystem
func (ns *Namespace) Unmount(path string) error

// List all mounts
func (ns *Namespace) Mounts() []Mount

type Mount struct {
    Path string
    FS   FS
}
```

### Bind Operations

```go
// Bind oldpath to newpath
func (ns *Namespace) Bind(oldpath, newpath string, flags BindFlag) error

// Remove a binding
func (ns *Namespace) Unbind(path string) error

// List all bindings
func (ns *Namespace) Bindings() []Binding

type Binding struct {
    From string
    To   string
}

// Bind flags
const (
    BIND_REPLACE BindFlag = 1 << iota
    BIND_BEFORE
    BIND_AFTER
)
```

## File Operations

### Opening Files

```go
// Open file for reading
file, err := ns.Open("/path/to/file")
if err != nil {
    if errors.Is(err, fs.ErrNotExist) {
        // File doesn't exist
    }
    return err
}
defer file.Close()

// Open with create
file, err := ns.OpenFile("/path/to/file", os.O_CREATE|os.O_WRONLY, 0644)
```

### Reading Files

```go
// Read all at once
data, err := fs.ReadFile(ns, "/path/to/file")

// Read in chunks
file, _ := ns.Open("/path/to/file")
buffer := make([]byte, 1024)
for {
    n, err := file.Read(buffer)
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    process(buffer[:n])
}

// Read with seek
file.Seek(100, io.SeekStart)  // Absolute position
file.Seek(50, io.SeekCurrent) // Relative to current
file.Seek(-10, io.SeekEnd)     // Relative to end
```

### Writing Files

```go
// Write all at once
err := ns.WriteFile("/path/to/file", []byte("content"), 0644)

// Write incrementally
file, _ := ns.Create("/path/to/file")
file.Write([]byte("line 1\n"))
file.Write([]byte("line 2\n"))
file.Sync() // Force to disk
file.Close()

// Append to file
file, _ := ns.OpenFile("/path/to/file", os.O_APPEND|os.O_WRONLY, 0)
file.Write([]byte("appended\n"))
```

### Directory Operations

```go
// Create directory
err := ns.Mkdir("/path/to/dir", 0755)

// Create nested directories
err := ns.MkdirAll("/path/to/nested/dir", 0755)

// List directory
entries, err := fs.ReadDir(ns, "/path/to/dir")
for _, entry := range entries {
    fmt.Printf("%s (%v)\n", entry.Name(), entry.IsDir())
}

// Read directory from file
dir, _ := ns.Open("/path/to/dir")
entries, err := dir.Readdir(-1) // -1 means all entries
```

## Path Operations

### Path Manipulation

```go
import "path"

// Clean path
clean := path.Clean("/path//to/../file") // "/path/file"

// Join paths
joined := path.Join("/base", "sub", "file.txt") // "/base/sub/file.txt"

// Split path
dir, file := path.Split("/path/to/file.txt") // "/path/to/", "file.txt"

// Get extension
ext := path.Ext("file.txt") // ".txt"

// Get base name
base := path.Base("/path/to/file.txt") // "file.txt"

// Get directory
dir := path.Dir("/path/to/file.txt") // "/path/to"
```

### Path Resolution

```go
// Resolve relative to working directory
resolved, err := ns.Resolve("relative/path")

// Check if path exists
_, err := ns.Stat("/path/to/check")
exists := err == nil

// Check if path is directory
info, _ := ns.Stat("/path")
isDir := info.IsDir()
```

## Advanced Operations

### File Locking

```go
// Advisory locking (if supported)
type LockableFile interface {
    File
    Lock() error
    Unlock() error
}

if lockable, ok := file.(LockableFile); ok {
    lockable.Lock()
    defer lockable.Unlock()
}
```

### Memory Mapping

```go
// Memory mapping (if supported)
type MappableFile interface {
    File
    Mmap(offset int64, length int) ([]byte, error)
    Munmap([]byte) error
}

if mappable, ok := file.(MappableFile); ok {
    data, err := mappable.Mmap(0, 1024)
    // Use data directly
    mappable.Munmap(data)
}
```

### Extended Attributes

```go
// Extended attributes (if supported)
type XAttrFile interface {
    File
    GetXAttr(name string) ([]byte, error)
    SetXAttr(name string, value []byte) error
    ListXAttr() ([]string, error)
}
```

## Error Handling

### Standard Errors

```go
import "io/fs"

// Check specific errors
if errors.Is(err, fs.ErrNotExist) {
    // File not found
}
if errors.Is(err, fs.ErrPermission) {
    // Permission denied
}
if errors.Is(err, fs.ErrExist) {
    // File already exists
}
if errors.Is(err, fs.ErrInvalid) {
    // Invalid operation
}
```

### Path Errors

```go
var pathErr *fs.PathError
if errors.As(err, &pathErr) {
    fmt.Printf("Operation: %s\n", pathErr.Op)
    fmt.Printf("Path: %s\n", pathErr.Path)
    fmt.Printf("Error: %v\n", pathErr.Err)
}
```

## Performance Considerations

### Buffered I/O

```go
import "bufio"

// Buffered reading
file, _ := ns.Open("/large/file")
reader := bufio.NewReader(file)
line, err := reader.ReadString('\n')

// Buffered writing
file, _ := ns.Create("/output")
writer := bufio.NewWriter(file)
writer.WriteString("buffered\n")
writer.Flush()
```

### Bulk Operations

```go
// Use ReadFile for small files
data, _ := fs.ReadFile(ns, "/small/file")

// Use streaming for large files
file, _ := ns.Open("/large/file")
io.Copy(dst, file)
```

### Caching

```go
// File info caching
type CachedFS struct {
    FS
    cache map[string]FileInfo
}

func (c *CachedFS) Stat(name string) (FileInfo, error) {
    if info, ok := c.cache[name]; ok {
        return info, nil
    }
    info, err := c.FS.Stat(name)
    if err == nil {
        c.cache[name] = info
    }
    return info, err
}
```

## Best Practices

1. **Always close files** - Use defer for cleanup
2. **Check errors** - Don't ignore error returns
3. **Use appropriate buffer sizes** - Balance memory vs. performance
4. **Handle partial writes** - Write may not write all bytes
5. **Use path package** - For portable path manipulation
6. **Validate paths** - Prevent directory traversal attacks
7. **Set appropriate permissions** - Follow principle of least privilege

## Examples

### Copy File

```go
func copyFile(ns *Namespace, src, dst string) error {
    srcFile, err := ns.Open(src)
    if err != nil {
        return err
    }
    defer srcFile.Close()
    
    dstFile, err := ns.Create(dst)
    if err != nil {
        return err
    }
    defer dstFile.Close()
    
    _, err = io.Copy(dstFile, srcFile)
    return err
}
```

### Walk Directory Tree

```go
func walkDir(ns *Namespace, root string, fn func(path string, info FileInfo) error) error {
    entries, err := fs.ReadDir(ns, root)
    if err != nil {
        return err
    }
    
    for _, entry := range entries {
        path := path.Join(root, entry.Name())
        info, err := entry.Info()
        if err != nil {
            return err
        }
        
        if err := fn(path, info); err != nil {
            return err
        }
        
        if entry.IsDir() {
            if err := walkDir(ns, path, fn); err != nil {
                return err
            }
        }
    }
    
    return nil
}
```