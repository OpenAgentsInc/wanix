# Debugging

This guide covers debugging techniques and tools for Wanix development, including both native and WebAssembly debugging.

## Overview

Debugging Wanix involves different approaches depending on the context:
- **Native debugging** for the Go executable
- **Browser debugging** for WebAssembly and web integration
- **Logging and tracing** for distributed debugging
- **Performance profiling** for optimization

## Native Debugging

### Using Delve (Go Debugger)

Install Delve:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

Debug Wanix:
```bash
# Start debugger
dlv debug ./cmd/wanix -- serve

# Or attach to running process
dlv attach $(pgrep wanix)
```

Common Delve commands:
```
(dlv) break main.main           # Set breakpoint
(dlv) continue                  # Run to breakpoint
(dlv) next                      # Step over
(dlv) step                      # Step into
(dlv) print varname             # Print variable
(dlv) locals                    # Show local variables
(dlv) stack                     # Show stack trace
(dlv) goroutines               # List goroutines
```

### VS Code Debugging

`.vscode/launch.json`:
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Wanix",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/wanix",
            "args": ["serve"],
            "env": {
                "WANIX_DEBUG": "1"
            }
        },
        {
            "name": "Attach to Process",
            "type": "go",
            "request": "attach",
            "mode": "local",
            "processId": "${command:pickProcess}"
        }
    ]
}
```

## Browser Debugging

### Chrome DevTools

1. **Open DevTools**: F12 or right-click → Inspect
2. **Sources Panel**: Debug JavaScript and WASM
3. **Console**: View logs and errors
4. **Network**: Monitor requests
5. **Performance**: Profile execution

### WebAssembly Debugging

Enable WASM debugging:
```bash
# Build with source maps
GOOS=js GOARCH=wasm go build -gcflags="-N -l" -o wanix.wasm ./wasm

# Or with TinyGo
tinygo build -no-debug=false -o wanix.wasm ./wasm
```

Browser setup:
1. Chrome: Enable "WebAssembly Debugging" in DevTools Settings
2. Firefox: Automatic with DWARF info

### Console Logging

Add debug logging:
```go
//go:build wasm

package main

import "syscall/js"

func debugLog(msg string, args ...interface{}) {
    console := js.Global().Get("console")
    console.Call("log", append([]interface{}{msg}, args...)...)
}

func debugDir(obj js.Value) {
    js.Global().Get("console").Call("dir", obj)
}
```

## Logging and Tracing

### Structured Logging

Use structured logging for better debugging:

```go
package myservice

import (
    "log/slog"
    "os"
)

var logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

func (s *Service) Open(name string) (fs.File, error) {
    logger.Debug("opening file",
        "service", "myservice",
        "path", name,
        "caller", getCaller(),
    )
    
    f, err := s.doOpen(name)
    if err != nil {
        logger.Error("open failed",
            "path", name,
            "error", err,
        )
    }
    
    return f, err
}
```

### Debug Mode

Enable debug mode with environment variable:

```go
var debug = os.Getenv("WANIX_DEBUG") != ""

func debugf(format string, args ...interface{}) {
    if debug {
        log.Printf("[DEBUG] "+format, args...)
    }
}
```

### Trace Execution

Add execution tracing:

```go
func trace(name string) func() {
    if !debug {
        return func() {}
    }
    
    start := time.Now()
    log.Printf("[TRACE] %s: start", name)
    
    return func() {
        log.Printf("[TRACE] %s: done (%v)", name, time.Since(start))
    }
}

func (s *Service) ComplexOperation() error {
    defer trace("ComplexOperation")()
    
    // Operation implementation...
}
```

## Common Issues and Solutions

### 1. WASM Panics

Problem: WASM module panics with unclear error

Solution:
```go
// Add panic recovery
func init() {
    if runtime.GOOS == "js" {
        defer func() {
            if r := recover(); r != nil {
                js.Global().Get("console").Call("error", "PANIC:", r)
                debug.PrintStack()
            }
        }()
    }
}
```

### 2. Deadlocks

Problem: Application hangs

Debug approach:
```go
// Add deadlock detection
import "github.com/sasha-s/go-deadlock"

// Replace sync.Mutex with deadlock.Mutex
type Service struct {
    mu deadlock.Mutex  // Will detect deadlocks
}

// Or use runtime detection
func detectDeadlock() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        buf := make([]byte, 1<<20)
        n := runtime.Stack(buf, true)
        log.Printf("=== GOROUTINE DUMP ===\n%s\n", buf[:n])
    }
}
```

### 3. Memory Leaks

Problem: Memory usage grows over time

Debug approach:
```go
import (
    "runtime"
    "runtime/pprof"
)

func writeMemProfile() {
    f, _ := os.Create("mem.prof")
    defer f.Close()
    
    runtime.GC()
    pprof.WriteHeapProfile(f)
}

// Analyze with: go tool pprof mem.prof
```

### 4. File Descriptor Leaks

Problem: Running out of file descriptors

Debug approach:
```go
type TrackedFile struct {
    fs.File
    path      string
    openStack string
}

func (s *Service) Open(name string) (fs.File, error) {
    f, err := s.fs.Open(name)
    if err != nil {
        return nil, err
    }
    
    return &TrackedFile{
        File:      f,
        path:      name,
        openStack: string(debug.Stack()),
    }, nil
}

func (f *TrackedFile) Close() error {
    log.Printf("Closing file %s opened at:\n%s", f.path, f.openStack)
    return f.File.Close()
}
```

## Performance Profiling

### CPU Profiling

```go
import "runtime/pprof"

func startCPUProfile() {
    f, _ := os.Create("cpu.prof")
    pprof.StartCPUProfile(f)
    
    // Run your code
    
    pprof.StopCPUProfile()
    f.Close()
}

// Analyze:
// go tool pprof cpu.prof
// (pprof) top
// (pprof) web
```

### Memory Profiling

```go
func captureMemProfile() {
    f, _ := os.Create("mem.prof")
    defer f.Close()
    
    runtime.GC()
    pprof.WriteHeapProfile(f)
}

// Analyze:
// go tool pprof -alloc_space mem.prof
// go tool pprof -inuse_space mem.prof
```

### Browser Performance

Use Chrome DevTools Performance tab:
1. Start recording
2. Perform operations
3. Stop recording
4. Analyze flame graph

## Debug Utilities

### File System Debugging

```go
// fsutil/debug.go
type DebugFS struct {
    fs.FS
    prefix string
}

func (d *DebugFS) Open(name string) (fs.File, error) {
    log.Printf("%s: Open(%q)", d.prefix, name)
    
    f, err := d.FS.Open(name)
    if err != nil {
        log.Printf("%s: Open(%q) error: %v", d.prefix, name, err)
        return nil, err
    }
    
    return &DebugFile{File: f, path: name, fs: d.prefix}, nil
}

type DebugFile struct {
    fs.File
    path string
    fs   string
}

func (f *DebugFile) Read(p []byte) (int, error) {
    n, err := f.File.Read(p)
    log.Printf("%s: Read(%q) = %d, %v", f.fs, f.path, n, err)
    return n, err
}
```

### VFS Debugging

```go
// Enable VFS operation logging
func DebugNamespace(ns *vfs.Namespace) *vfs.Namespace {
    return &debugNamespace{
        Namespace: ns,
        ops:       make(map[string]int),
    }
}

type debugNamespace struct {
    *vfs.Namespace
    ops map[string]int
    mu  sync.Mutex
}

func (d *debugNamespace) Open(path string) (fs.File, error) {
    d.mu.Lock()
    d.ops["Open"]++
    d.mu.Unlock()
    
    start := time.Now()
    f, err := d.Namespace.Open(path)
    
    log.Printf("VFS Open(%q): %v (%v)", path, err, time.Since(start))
    return f, err
}
```

## Interactive Debugging

### REPL for Testing

```go
// debug/repl.go
func StartREPL(ns *vfs.Namespace) {
    scanner := bufio.NewScanner(os.Stdin)
    
    fmt.Println("Wanix Debug REPL")
    fmt.Println("Commands: ls, cat, write, stat, quit")
    
    for {
        fmt.Print("> ")
        if !scanner.Scan() {
            break
        }
        
        parts := strings.Fields(scanner.Text())
        if len(parts) == 0 {
            continue
        }
        
        switch parts[0] {
        case "ls":
            cmdLs(ns, parts[1:])
        case "cat":
            cmdCat(ns, parts[1:])
        case "write":
            cmdWrite(ns, parts[1:])
        case "stat":
            cmdStat(ns, parts[1:])
        case "quit":
            return
        default:
            fmt.Println("Unknown command:", parts[0])
        }
    }
}
```

### Browser Console Helpers

```javascript
// Add debug helpers to browser
window.wanixDebug = {
    async listFiles(path = '/') {
        const entries = await wanix.readdir(path);
        console.table(entries);
    },
    
    async readFile(path) {
        const data = await wanix.readFile(path);
        console.log(data);
    },
    
    async traceOpen(path) {
        console.time('open');
        const file = await wanix.open(path);
        console.timeEnd('open');
        return file;
    },
    
    dumpNamespace() {
        // Implementation depends on internal structure
        console.dir(wanix.namespace);
    }
};
```

## Production Debugging

### Debug Endpoints

Add debug endpoints for production:

```go
func (s *Server) addDebugHandlers() {
    http.HandleFunc("/debug/vars", expvar.Handler())
    
    http.HandleFunc("/debug/pprof/", pprof.Index)
    http.HandleFunc("/debug/pprof/heap", pprof.Handler("heap"))
    http.HandleFunc("/debug/pprof/profile", pprof.Profile)
    
    http.HandleFunc("/debug/vfs", s.handleVFSDebug)
    http.HandleFunc("/debug/tasks", s.handleTasksDebug)
}
```

### Metrics Collection

```go
var (
    fileOpens = expvar.NewInt("file_opens")
    fileReads = expvar.NewInt("file_reads")
    errors    = expvar.NewMap("errors")
)

func (s *Service) Open(name string) (fs.File, error) {
    fileOpens.Add(1)
    
    f, err := s.doOpen(name)
    if err != nil {
        errors.Add(err.Error(), 1)
    }
    
    return f, err
}
```

## Tips and Tricks

1. **Binary Search for Bugs**: Use git bisect to find when bug was introduced
2. **Minimal Reproducers**: Create smallest possible test case
3. **Compare Working/Broken**: Diff logs between working and broken states
4. **Rubber Duck Debugging**: Explain the problem out loud
5. **Take Breaks**: Fresh perspective often reveals solutions

## Next Steps

1. Set up debugging environment for your editor
2. Add debug logging to your code
3. Learn browser DevTools for WASM debugging
4. Create debug utilities for your use cases
5. Document debugging procedures for your team