# Capabilities Subsystem

The capabilities subsystem implements Wanix's security model, providing controlled access to system resources through file-based interfaces.

## Overview

Capabilities in Wanix are:
- **Unforgeable tokens** that grant access to resources
- **File services** that can be allocated and managed
- **Composable** building blocks for system functionality
- **The only way** to access privileged operations

## Architecture

```
┌─────────────────────────────────────┐
│          /cap Directory             │
│  ├── new/                          │
│  │   ├── tarfs                     │
│  │   ├── tmpfs                     │
│  │   ├── loopback                  │
│  │   └── ...                       │
│  └── <id>/                         │
│      ├── ctl                       │
│      ├── type                      │
│      └── ...                       │
└─────────────────────────────────────┘
```

## Core Implementation (`cap/service.go`)

### Capability Service

The main capability service manages all capabilities:

```go
type Service struct {
    caps     map[string]Capability
    types    map[string]CapabilityType
    nextID   int
    mu       sync.RWMutex
}

// Capability interface
type Capability interface {
    fs.FS
    Type() string
    Close() error
}

// Capability type factory
type CapabilityType interface {
    Name() string
    New() (Capability, error)
}
```

### The Clone Dance Pattern

The Plan 9-inspired pattern for allocating capabilities:

```bash
# 1. Read the clone file to allocate
id=$(cat /cap/new/tmpfs)

# 2. Access the new capability
echo "Hello" > /cap/$id/file.txt

# 3. Use control file for operations
echo "snapshot" > /cap/$id/ctl
```

Implementation:
```go
func (s *Service) Open(name string) (fs.File, error) {
    // Handle clone files
    if strings.HasPrefix(name, "new/") {
        capType := strings.TrimPrefix(name, "new/")
        return s.newCloneFile(capType), nil
    }
    
    // Route to capability instance
    parts := strings.Split(name, "/")
    cap := s.caps[parts[0]]
    return cap.Open(strings.Join(parts[1:], "/"))
}
```

## Built-in Capabilities

### 1. TarFS Capability (`cap/tarfs.go`)

Creates read-only filesystems from TAR archives:

```go
type TarFS struct {
    id     string
    fs     fs.FS
    source io.Reader
}

// Usage example
func createTarFS(data []byte) {
    // Allocate capability
    id := readFile("/cap/new/tarfs")
    
    // Write TAR data
    writeFile(fmt.Sprintf("/cap/%s/data", id), data)
    
    // Mount the filesystem
    bind(fmt.Sprintf("/cap/%s/fs", id), "/mnt/archive")
}
```

Features:
- Lazy extraction
- Memory efficient
- Supports all TAR formats
- Read-only access

### 2. TmpFS Capability

Creates temporary in-memory filesystems:

```go
type TmpFS struct {
    id   string
    fs   *fskit.MemFS
    size int64
}

// Control commands
func (t *TmpFS) Control(cmd string, args []string) error {
    switch cmd {
    case "limit":
        return t.setLimit(args[0])
    case "clear":
        return t.clear()
    case "snapshot":
        return t.snapshot()
    }
}
```

Features:
- Fast in-memory storage
- Size limits
- Snapshot support
- Automatic cleanup

### 3. Loopback Capability (`cap/loopback.go`)

Exposes existing namespace portions:

```go
type Loopback struct {
    id    string
    ns    *vfs.Namespace
    root  string
}

// Create filtered view
func (l *Loopback) SetFilter(filter func(path string) bool) {
    l.filter = filter
}
```

Use cases:
- Namespace isolation
- Filtered filesystem views
- Testing and debugging
- Security boundaries

### 4. Resource Capability (`cap/resource.go`)

Generic resource management:

```go
type Resource struct {
    id       string
    name     string
    data     interface{}
    refcount int32
}

// Reference counting
func (r *Resource) Acquire() {
    atomic.AddInt32(&r.refcount, 1)
}

func (r *Resource) Release() {
    if atomic.AddInt32(&r.refcount, -1) == 0 {
        r.cleanup()
    }
}
```

## Web-Specific Capabilities

### PickerFS (`web/fsa/fs.go`)

Browser file picker integration:

```go
type PickerFS struct {
    id     string
    handle js.Value  // FileSystemDirectoryHandle
}

func (p *PickerFS) Mount() error {
    // Show directory picker
    promise := js.Global().Call("showDirectoryPicker")
    
    // Wait for user selection
    handle := await(promise)
    
    // Create filesystem from handle
    p.fs = &fsaFS{handle: handle}
    return nil
}
```

### Worker Capability (`web/worker/service.go`)

Web Worker management:

```go
type Worker struct {
    id     string
    worker js.Value
    status string
}

func (w *Worker) Start(script string) error {
    // Create web worker
    w.worker = js.Global().Get("Worker").New(script)
    
    // Set up message handling
    w.worker.Set("onmessage", js.FuncOf(w.onMessage))
    
    w.status = "running"
    return nil
}
```

## Creating Custom Capabilities

### Basic Structure

```go
package mycap

type MyCapability struct {
    id   string
    data map[string]interface{}
}

// Implement Capability interface
func (m *MyCapability) Type() string {
    return "mycap"
}

func (m *MyCapability) Open(name string) (fs.File, error) {
    switch name {
    case "ctl":
        return m.openControl()
    case "status":
        return m.openStatus()
    default:
        return nil, fs.ErrNotExist
    }
}

func (m *MyCapability) Close() error {
    // Cleanup resources
    return nil
}
```

### Registration

```go
// Register with capability service
func init() {
    cap.Register("mycap", &MyCapabilityType{})
}

type MyCapabilityType struct{}

func (t *MyCapabilityType) Name() string {
    return "mycap"
}

func (t *MyCapabilityType) New() (cap.Capability, error) {
    return &MyCapability{
        id:   generateID(),
        data: make(map[string]interface{}),
    }, nil
}
```

## Security Model

### Capability Properties

1. **Unforgeable**: Can't create capabilities without proper allocation
2. **Transferable**: Can be passed between processes via namespace
3. **Revocable**: Can be removed from namespace
4. **Composable**: Combine capabilities for complex behaviors

### Access Control

```go
// Only processes with capability in namespace can access
func checkAccess(ns *vfs.Namespace, capPath string) bool {
    _, err := ns.Stat(capPath)
    return err == nil
}

// Revoke by unbinding
func revokeAccess(ns *vfs.Namespace, capPath string) {
    ns.Unbind(capPath)
}
```

### Principle of Least Privilege

```go
// Give process only needed capabilities
func setupRestrictedNamespace() *vfs.Namespace {
    ns := vfs.NewNamespace()
    
    // Only file access, no network
    ns.Mount(opfs, "/storage")
    
    // No DOM access
    // No worker creation
    // No system capabilities
    
    return ns
}
```

## Capability Patterns

### 1. Delegation Pattern

```go
// Parent creates capability
parentCap := allocateCapability("resource")

// Bind to child's namespace
childNS.Bind(fmt.Sprintf("/cap/%s", parentCap.ID), "/resource")

// Child can now use, but not create new ones
```

### 2. Revocation Pattern

```go
// Time-limited capability
type ExpiringCap struct {
    cap     Capability
    expires time.Time
}

func (e *ExpiringCap) Open(name string) (fs.File, error) {
    if time.Now().After(e.expires) {
        return nil, errors.New("capability expired")
    }
    return e.cap.Open(name)
}
```

### 3. Composition Pattern

```go
// Combine multiple capabilities
type CompositeCap struct {
    read  Capability
    write Capability
}

func (c *CompositeCap) Open(name string) (fs.File, error) {
    if strings.HasPrefix(name, "read/") {
        return c.read.Open(strings.TrimPrefix(name, "read/"))
    }
    if strings.HasPrefix(name, "write/") {
        return c.write.Open(strings.TrimPrefix(name, "write/"))
    }
    return nil, fs.ErrNotExist
}
```

## Best Practices

1. **Minimize Capability Scope**
   - Create focused, single-purpose capabilities
   - Avoid "super capabilities" with too much power

2. **Clear Lifecycle Management**
   - Always clean up resources in Close()
   - Use reference counting for shared resources

3. **Fail Securely**
   - Default to denying access
   - Validate all inputs
   - Clear error messages

4. **Document Control Commands**
   - List all commands in control file
   - Provide clear command syntax
   - Return helpful error messages

## Performance Considerations

### Efficient Allocation

```go
// Pool capability objects
var capPool = sync.Pool{
    New: func() interface{} {
        return &Capability{}
    },
}

func allocateCapability() *Capability {
    cap := capPool.Get().(*Capability)
    cap.reset()
    return cap
}
```

### Lazy Initialization

```go
type LazyCapability struct {
    init sync.Once
    res  resource
}

func (l *LazyCapability) getResource() resource {
    l.init.Do(func() {
        l.res = expensiveInit()
    })
    return l.res
}
```

## Future Enhancements

- **Capability Inheritance**: Hierarchical capabilities
- **Audit Logging**: Track all capability usage
- **Remote Capabilities**: Network-transparent capabilities
- **Capability Persistence**: Save/restore capability state
- **Fine-grained Permissions**: More granular access control