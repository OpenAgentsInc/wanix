# VFS Design

The Virtual Filesystem (VFS) is the heart of Wanix, providing a unified interface to diverse resources while enabling per-process namespaces.

## Overview

The Wanix VFS is not just an in-memory filesystem - it's a sophisticated routing layer that:
- Delegates operations to appropriate file services
- Manages namespace composition and unions
- Handles bind mounts and redirections
- Provides process isolation through namespace separation

## Architecture

```
┌─────────────────────────────────────────────┐
│             User Process                    │
│         (read, write, open, etc.)          │
└───────────────────┬─────────────────────────┘
                    │
┌───────────────────▼─────────────────────────┐
│              VFS Layer                      │
│  • Path resolution                          │
│  • Namespace lookup                         │
│  • Union handling                           │
│  • Bind mount resolution                    │
└───────────────────┬─────────────────────────┘
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
┌───────────────┐       ┌───────────────┐
│ File Service  │       │ File Service  │
│   (OPFS)      │       │   (DOM)       │
└───────────────┘       └───────────────┘
```

## Core Components

### 1. Namespace (`vfs/vfs.go`)

Each namespace maintains:
- **Mount table**: Maps paths to file services
- **Bind table**: Tracks bind mount redirections  
- **Union sets**: Manages unified directory views
- **Parent reference**: For inheritance

Key operations:
```go
type Namespace interface {
    Bind(oldpath, newpath string, flags BindFlag) error
    Unbind(path string) error
    Walk(path string) (File, error)
    Stat(path string) (FileInfo, error)
}
```

### 2. Path Resolution

Path resolution follows these steps:

1. **Normalize path**: Convert to absolute, resolve `.` and `..`
2. **Check binds**: Look for bind mount redirections
3. **Find mount**: Locate longest matching mount point
4. **Handle unions**: If union mount, try each member
5. **Delegate**: Pass remaining path to file service

Example flow:
```
Path: /web/dom/new/iframe
1. Normalize: /web/dom/new/iframe
2. No binds match
3. Mount found: /web -> WebFS
4. Delegate: dom/new/iframe to WebFS
5. WebFS routes: dom -> DOMService
6. DOMService handles: new/iframe
```

### 3. Union Filesystems

Unions allow multiple directories to appear as one:

```go
// Multiple binds to same target create union
ns.Bind("/dir1", "/union", 0)
ns.Bind("/dir2", "/union", BIND_AFTER)
// Now /union shows contents of both dirs
```

Union behavior:
- **File precedence**: First bind wins for files
- **Directory merging**: All subdirs are merged
- **Write routing**: Writes go to first writable member
- **Transparency**: Union is invisible to applications

### 4. Special Mounts

Wanix includes several special mount types:

**Root mount (#)**: Provides built-in services
```
#null     -> Null device
#dot      -> Current namespace
#shell    -> Shell root filesystem
```

**Device mounts**: Hardware/pseudo devices
```
/dev/null     -> Discard data
/dev/random   -> Random bytes
/dev/cons     -> Console device
```

**Synthetic mounts**: Generated content
```
/proc         -> Process information
/env          -> Environment variables
```

## Implementation Details

### File Interface (`fs/file.go`)

All file services implement the core File interface:

```go
type File interface {
    io.Reader
    io.Writer
    io.Closer
    io.Seeker
    
    Readdir(n int) ([]FileInfo, error)
    Stat() (FileInfo, error)
    Sync() error
    Truncate(size int64) error
}
```

### FileSystem Interface (`fs/fskit/node.go`)

File services provide filesystem operations:

```go
type FS interface {
    Open(name string) (File, error)
    Create(name string) (File, error)
    Mkdir(name string, perm FileMode) error
    Remove(name string) error
    Rename(oldname, newname string) error
    Stat(name string) (FileInfo, error)
}
```

### VFS Operations

Key VFS operations in `vfs/vfs.go`:

**Walk**: Traverse namespace to find file
```go
func (ns *namespace) Walk(path string) (File, error) {
    // Resolve binds
    if target, ok := ns.binds[path]; ok {
        path = target
    }
    
    // Find mount point
    mnt := ns.findMount(path)
    
    // Delegate to file service
    return mnt.fs.Open(path[len(mnt.point):])
}
```

**Bind**: Create namespace binding
```go
func (ns *namespace) Bind(old, new string, flags int) error {
    // Handle union if target exists
    if ns.exists(new) && flags&BIND_AFTER != 0 {
        ns.addUnion(new, old)
    } else {
        ns.binds[new] = old
    }
    return nil
}
```

## File Service Integration

File services integrate with VFS by implementing standard interfaces:

### Example: MemFS (`fs/fskit/memfs.go`)

```go
type memFS struct {
    root *memNode
    mu   sync.RWMutex
}

func (fs *memFS) Open(name string) (File, error) {
    fs.mu.RLock()
    defer fs.mu.RUnlock()
    
    node := fs.walk(name)
    if node == nil {
        return nil, ErrNotExist
    }
    
    return &memFile{node: node}, nil
}
```

### Example: UnionFS (`fs/fskit/unionfs.go`)

```go
type unionFS struct {
    members []FS
}

func (fs *unionFS) Open(name string) (File, error) {
    // Try each member in order
    for _, member := range fs.members {
        f, err := member.Open(name)
        if err == nil {
            return f, nil
        }
    }
    return nil, ErrNotExist
}
```

## Advanced Features

### 1. Lazy Binding

Bindings can reference non-existent sources:
```bash
# Bind future mount point
bind /future/mount /current/path

# Later when /future/mount exists, binding activates
```

### 2. Recursive Binds

Entire subtrees can be rebound:
```bash
# Rebind entire web subtree
bind /web /isolated/web
```

### 3. Bind Flags

Control binding behavior:
- `BIND_REPLACE`: Replace existing target
- `BIND_AFTER`: Add to union after existing
- `BIND_BEFORE`: Add to union before existing

### 4. Namespace Inheritance

Child processes inherit parent namespace with copy-on-write semantics:
```go
child := parent.Fork()
child.Bind("/private", "/data", 0)  // Only affects child
```

## Performance Considerations

1. **Path Caching**: Frequently accessed paths are cached
2. **Mount Table**: Sorted for binary search efficiency
3. **Lazy Evaluation**: Unions evaluated only when accessed
4. **Copy-on-Write**: Namespace modifications are COW

## Security Properties

The VFS enforces security through:

1. **Namespace Isolation**: Processes can't escape their namespace
2. **No Path Traversal**: `..` can't escape namespace root
3. **Capability Enforcement**: No access without file service in namespace
4. **Immutable Bindings**: Some system bindings can't be modified

## Future Directions

- **Distributed VFS**: Network-transparent file services
- **Persistent Namespaces**: Save/restore namespace configurations
- **Fine-grained Permissions**: Per-file access controls
- **Performance Optimizations**: Improved caching and indexing