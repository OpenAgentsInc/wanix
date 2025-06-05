# Creating File Services

File services are the fundamental building blocks of Wanix. This guide shows how to create your own file services to extend Wanix's capabilities.

## Overview

A file service in Wanix:
- Implements the `fs.FS` interface
- Exposes functionality through file operations
- Can be mounted into any namespace
- Follows Plan 9-inspired patterns

## Basic File Service

### Minimal Implementation

```go
package myservice

import (
    "io/fs"
    "github.com/tractordev/wanix/fs/fskit"
)

type MyService struct {
    data map[string]string
}

func New() *MyService {
    return &MyService{
        data: make(map[string]string),
    }
}

// Implement fs.FS interface
func (s *MyService) Open(name string) (fs.File, error) {
    switch name {
    case "hello":
        return fskit.NewReadFile([]byte("Hello, World!")), nil
    case "ctl":
        return s.openControl(), nil
    default:
        return nil, fs.ErrNotExist
    }
}
```

### Control File Pattern

Most file services include a control file for operations:

```go
func (s *MyService) openControl() fs.File {
    return fskit.NewControlFile(s, []fskit.Command{
        {
            Name:  "set",
            Usage: "set <key> <value>",
            Func:  s.cmdSet,
        },
        {
            Name:  "get", 
            Usage: "get <key>",
            Func:  s.cmdGet,
        },
    })
}

func (s *MyService) cmdSet(args []string) ([]byte, error) {
    if len(args) != 2 {
        return nil, errors.New("usage: set <key> <value>")
    }
    s.data[args[0]] = args[1]
    return []byte("OK\n"), nil
}
```

## Advanced Patterns

### Clone Pattern (Capabilities)

For services that allocate resources:

```go
type ResourceService struct {
    resources map[string]*Resource
    mu        sync.RWMutex
}

func (s *ResourceService) Open(name string) (fs.File, error) {
    // Handle clone file
    if name == "new" {
        return s.openNew(), nil
    }
    
    // Handle resource directories
    parts := strings.Split(name, "/")
    if len(parts) >= 1 {
        s.mu.RLock()
        res, ok := s.resources[parts[0]]
        s.mu.RUnlock()
        
        if !ok {
            return nil, fs.ErrNotExist
        }
        
        // Delegate to resource
        subpath := strings.Join(parts[1:], "/")
        return res.Open(subpath)
    }
    
    return nil, fs.ErrNotExist
}

func (s *ResourceService) openNew() fs.File {
    return fskit.NewFuncFile(func() ([]byte, error) {
        // Allocate new resource
        res := &Resource{
            ID: generateID(),
        }
        
        s.mu.Lock()
        s.resources[res.ID] = res
        s.mu.Unlock()
        
        return []byte(res.ID + "\n"), nil
    })
}
```

### Dynamic Directory Listing

For services with dynamic content:

```go
func (s *MyService) Open(name string) (fs.File, error) {
    if name == "." || name == "" {
        return s.openRoot(), nil
    }
    // ... handle other files
}

func (s *MyService) openRoot() fs.File {
    return fskit.NewDirFile(func() ([]fs.DirEntry, error) {
        entries := []fs.DirEntry{
            fskit.NewDirEntry("ctl", false),
            fskit.NewDirEntry("status", false),
        }
        
        // Add dynamic entries
        s.mu.RLock()
        for id := range s.resources {
            entries = append(entries, fskit.NewDirEntry(id, true))
        }
        s.mu.RUnlock()
        
        return entries, nil
    })
}
```

### Streaming Files

For real-time data streams:

```go
type StreamFile struct {
    ch     chan []byte
    buffer bytes.Buffer
    mu     sync.Mutex
}

func (f *StreamFile) Read(p []byte) (int, error) {
    f.mu.Lock()
    defer f.mu.Unlock()
    
    // Check buffer first
    if f.buffer.Len() > 0 {
        return f.buffer.Read(p)
    }
    
    // Wait for data
    select {
    case data := <-f.ch:
        f.buffer.Write(data)
        return f.buffer.Read(p)
    case <-time.After(time.Second):
        return 0, nil // No data available
    }
}
```

## File Service Toolkit

Wanix provides utilities in `fs/fskit/` for common patterns:

### Static Files

```go
// Single static file
file := fskit.NewReadFile([]byte("static content"))

// Writable file
file := fskit.NewMemFile()
file.Write([]byte("initial content"))
```

### Function Files

```go
// Dynamic content on each read
timeFile := fskit.NewFuncFile(func() ([]byte, error) {
    return []byte(time.Now().String()), nil
})

// With write support
echoFile := fskit.NewReadWriteFuncFile(
    func() ([]byte, error) {
        return lastEcho, nil
    },
    func(data []byte) error {
        lastEcho = data
        return nil
    },
)
```

### Directory Files

```go
// Static directory
dir := fskit.NewDir(map[string]fs.File{
    "readme.txt": fskit.NewReadFile([]byte("Hello")),
    "config.json": fskit.NewReadFile([]byte("{}")),
})

// Dynamic directory
dir := fskit.NewDirFile(func() ([]fs.DirEntry, error) {
    return getCurrentEntries(), nil
})
```

### MapFS

```go
// Create filesystem from map
mfs := fskit.MapFS{
    "file1.txt": &fskit.File{Data: []byte("content1")},
    "dir/file2.txt": &fskit.File{Data: []byte("content2")},
}
```

## Integration with Wanix

### As a Module

```go
package mymodule

import "github.com/tractordev/wanix"

type MyModule struct {
    service *MyService
}

func (m *MyModule) Name() string {
    return "myservice"
}

func (m *MyModule) Mount(ns *vfs.Namespace) error {
    m.service = myservice.New()
    return ns.Mount(m.service, "/myservice")
}

// Register during init
func init() {
    wanix.RegisterModule(&MyModule{})
}
```

### As a Capability

```go
package mycap

import "github.com/tractordev/wanix/cap"

type MyCapability struct {
    id string
    // ... capability state
}

func (c *MyCapability) Type() string {
    return "mycap"
}

// Register capability type
func init() {
    cap.Register("mycap", &MyCapabilityType{})
}
```

## Real-World Examples

### Timer Service

A complete timer service example:

```go
type TimerService struct {
    timers map[string]*Timer
    mu     sync.RWMutex
}

type Timer struct {
    ID       string
    Duration time.Duration
    Start    time.Time
    timer    *time.Timer
    done     chan struct{}
}

func (s *TimerService) Open(name string) (fs.File, error) {
    if name == "new" {
        return fskit.NewControlFile(s, []fskit.Command{
            {"create", "create <duration>", s.cmdCreate},
        }), nil
    }
    
    parts := strings.Split(name, "/")
    if len(parts) >= 2 {
        timer := s.getTimer(parts[0])
        if timer == nil {
            return nil, fs.ErrNotExist
        }
        
        switch parts[1] {
        case "ctl":
            return timer.openControl(), nil
        case "status":
            return timer.openStatus(), nil
        case "wait":
            return timer.openWait(), nil
        }
    }
    
    return nil, fs.ErrNotExist
}

func (t *Timer) openWait() fs.File {
    return fskit.NewFuncFile(func() ([]byte, error) {
        <-t.done
        return []byte("done\n"), nil
    })
}
```

### Event Stream Service

For event-driven applications:

```go
type EventService struct {
    subscribers map[string]*Subscriber
    events      chan Event
}

type Subscriber struct {
    ID     string
    Filter func(Event) bool
    ch     chan Event
}

func (s *Subscriber) Open(name string) (fs.File, error) {
    if name == "events" {
        return &EventFile{sub: s}, nil
    }
    return nil, fs.ErrNotExist
}

type EventFile struct {
    sub    *Subscriber
    buffer bytes.Buffer
}

func (f *EventFile) Read(p []byte) (int, error) {
    if f.buffer.Len() == 0 {
        // Wait for next event
        event := <-f.sub.ch
        data, _ := json.Marshal(event)
        f.buffer.Write(data)
        f.buffer.WriteByte('\n')
    }
    
    return f.buffer.Read(p)
}
```

## Best Practices

### 1. Error Handling

Always return appropriate errors:

```go
func (s *MyService) Open(name string) (fs.File, error) {
    if name == "" {
        return nil, fs.ErrInvalid
    }
    
    if !s.exists(name) {
        return nil, fs.ErrNotExist
    }
    
    if !s.hasPermission(name) {
        return nil, fs.ErrPermission
    }
    
    // ...
}
```

### 2. Concurrency Safety

Protect shared state:

```go
type SafeService struct {
    mu   sync.RWMutex
    data map[string]string
}

func (s *SafeService) Get(key string) string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.data[key]
}

func (s *SafeService) Set(key, value string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.data[key] = value
}
```

### 3. Resource Management

Clean up resources properly:

```go
type ResourceFile struct {
    resource *Resource
}

func (f *ResourceFile) Close() error {
    if f.resource != nil {
        f.resource.Release()
        f.resource = nil
    }
    return nil
}
```

### 4. Documentation

Document control commands clearly:

```go
var commands = []fskit.Command{
    {
        Name:  "start",
        Usage: "start [options...] - Start the service",
        Help: `Start the service with optional configuration.
Options:
  -p <port>    Port to listen on (default: 8080)
  -d           Enable debug mode`,
        Func: cmdStart,
    },
}
```

## Testing File Services

### Unit Tests

```go
func TestMyService(t *testing.T) {
    svc := New()
    
    // Test file operations
    f, err := svc.Open("hello")
    if err != nil {
        t.Fatal(err)
    }
    defer f.Close()
    
    data, err := io.ReadAll(f)
    if err != nil {
        t.Fatal(err)
    }
    
    if string(data) != "Hello, World!" {
        t.Errorf("got %q, want %q", data, "Hello, World!")
    }
}
```

### Integration Tests

```go
func TestServiceInNamespace(t *testing.T) {
    ns := vfs.NewNamespace()
    svc := New()
    
    err := ns.Mount(svc, "/test")
    if err != nil {
        t.Fatal(err)
    }
    
    // Test through VFS
    data, err := fs.ReadFile(ns, "/test/hello")
    if err != nil {
        t.Fatal(err)
    }
    
    // Verify result
    if string(data) != "Hello, World!" {
        t.Errorf("unexpected content: %s", data)
    }
}
```

## Debugging Tips

### Logging

```go
func (s *MyService) Open(name string) (fs.File, error) {
    if debug {
        log.Printf("MyService.Open(%q)", name)
    }
    
    // ... implementation
}
```

### Tracing

```go
type TracedFile struct {
    fs.File
    path string
}

func (f *TracedFile) Read(p []byte) (int, error) {
    n, err := f.File.Read(p)
    log.Printf("Read %q: %d bytes, err=%v", f.path, n, err)
    return n, err
}
```

## Next Steps

1. Study existing services in `web/`, `cap/`, and `task/`
2. Start with a simple read-only service
3. Add control file for operations
4. Implement resource allocation if needed
5. Write comprehensive tests
6. Document your service in code and README