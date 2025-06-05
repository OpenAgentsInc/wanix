# Wanix Quick Reference

A concise guide for developers working with the Wanix codebase.

## What is Wanix?

Wanix is a Plan 9-inspired virtual environment that runs in browsers and natively. Everything is a file, each process has its own namespace, and capabilities control access to resources.

## Key Concepts in 30 Seconds

1. **Everything is a file** - DOM, workers, processes, sockets - all accessed via filesystem
2. **Per-process namespaces** - Each process sees its own filesystem view
3. **Capabilities** - Resources accessed through file services in your namespace
4. **Two runtimes** - Runs WASI WebAssembly and x86 Linux binaries

## Project Structure

```
wanix/
├── cmd/wanix/     # CLI entry point
├── wanix.go       # Kernel initialization
├── vfs/           # Virtual filesystem (core!)
├── fs/            # Filesystem implementations
├── cap/           # Capability system
├── task/          # Process management
├── web/           # Browser integration
├── external/      # Dependencies (v86, Linux, WASI)
└── wasm/          # Browser module
```

## Common Development Tasks

### Building
```bash
make deps   # First time only (10-30 min)
make build  # Build everything
./wanix serve  # Run server
```

### Creating a File Service
```go
type MyService struct{}

func (s *MyService) Open(name string) (fs.File, error) {
    switch name {
    case "hello":
        return fskit.NewReadFile([]byte("Hello!")), nil
    case "ctl":
        return fskit.NewControlFile(s, commands), nil
    default:
        return nil, fs.ErrNotExist
    }
}
```

### Adding a Capability
```go
func init() {
    cap.Register("mytype", &MyCapabilityType{})
}

type MyCapabilityType struct{}

func (t *MyCapabilityType) New() (cap.Capability, error) {
    return &MyCapability{id: generateID()}, nil
}
```

### Testing
```bash
go test ./...              # Run all tests
go test -v ./fs/fskit      # Test specific package
go test -run TestName      # Run specific test
```

## File System Patterns

### Clone Pattern (Allocating Resources)
```bash
# Read clone file to allocate
id=$(cat /cap/new/tmpfs)
# Use the resource
echo "data" > /cap/$id/file.txt
```

### Control Files
```bash
# Send commands
echo "start /script.js" > /web/worker/$id/ctl
echo "append-child $child" > /web/dom/body/ctl
```

### Union Mounts
```bash
# Merge directories
bind /dir1 /union
bind /dir2 /union -a  # Append to union
```

## Important Code Locations

| Feature | Location |
|---------|----------|
| VFS core | `vfs/vfs.go` |
| Filesystem toolkit | `fs/fskit/` |
| Web APIs | `web/dom/`, `web/worker/` |
| WASI runtime | `external/wasi/` |
| Task management | `task/service.go` |
| Capability system | `cap/service.go` |

## Common Gotchas

1. **Always close files** - Use `defer file.Close()`
2. **Check errors** - Don't ignore error returns
3. **Path must be absolute** - Use `/path` not `path`
4. **Namespace isolation** - Process can only access what's in its namespace
5. **WASM size** - Use TinyGo for production builds

## Debugging Tips

### Native Debugging
```bash
# Use Delve
dlv debug ./cmd/wanix -- serve

# Or print debugging
export WANIX_DEBUG=1
./wanix serve
```

### Browser Debugging
1. Open Chrome DevTools
2. Enable WebAssembly debugging in settings
3. Use `console.log` from Go:
```go
js.Global().Get("console").Call("log", "debug message")
```

## Key APIs

### VFS Operations
```go
// Open file
file, err := ns.Open("/path/to/file")
defer file.Close()

// Read all
data, err := fs.ReadFile(ns, "/path")

// Create directory
err := ns.Mkdir("/newdir", 0755)
```

### Control Commands
```go
// Define commands
commands := []fskit.Command{
    {"start", "start <url>", cmdStart},
    {"stop", "stop", cmdStop},
}

// Create control file
ctl := fskit.NewControlFile(service, commands)
```

## Architecture Philosophy

1. **Simplicity** - Small, composable pieces
2. **Everything is a file** - Uniform interface
3. **Capability security** - No ambient authority
4. **Plan 9 inspired** - But pragmatic for the web

## Getting Help

1. Read the comprehensive docs in `/docs/`
2. Look at existing implementations
3. Ask in Discord
4. Check the video transcript for context

## Next Steps

- Explore existing file services in `web/` and `cap/`
- Try creating a simple file service
- Run the test suite and study test patterns
- Build something cool with Wanix!

Remember: In Wanix, if you can read/write files, you can do anything.