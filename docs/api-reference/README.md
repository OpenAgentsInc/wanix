# API Reference

Complete API documentation for Wanix, covering all public interfaces and file services.

## Overview

Wanix exposes all functionality through filesystem operations. This reference documents:
- Core VFS operations
- Built-in file services
- Control file commands
- JavaScript/WebAssembly APIs

## API Documentation

### [VFS API](./vfs-api.md)
Core filesystem operations:
- File operations (open, read, write, close)
- Directory operations (mkdir, readdir)
- Namespace operations (bind, mount)
- Path resolution and manipulation

### [File Services](./file-services.md)
Built-in services reference:
- System services (/cap, /task)
- Web services (/web/*)
- Storage services (OPFS, tmpfs)
- Device services

### [Control Files](./control-files.md)
Command reference for control files:
- Standard control patterns
- Service-specific commands
- Command syntax and responses
- Error handling

## Quick Reference

### Basic File Operations

```go
// Open a file
file, err := fs.Open("/path/to/file")

// Read data
data := make([]byte, 1024)
n, err := file.Read(data)

// Write data
n, err := file.Write([]byte("content"))

// Close file
err := file.Close()
```

### Namespace Operations

```bash
# Bind a file/directory
bind /source/path /target/path

# Create union mount
bind /dir1 /union
bind /dir2 /union -a  # append to union

# Remove binding
unbind /target/path
```

### Common Control Commands

```bash
# Capability allocation
id=$(cat /cap/new/tmpfs)

# Process control
echo "start /bin/app" > /task/$id/ctl
echo "kill 15" > /task/$id/ctl

# DOM manipulation
echo "append-child $child_id" > /web/dom/body/ctl
```

## Service Paths

### System Services

| Path | Description |
|------|-------------|
| `/cap` | Capability management |
| `/cap/new/*` | Capability allocation |
| `/cap/<id>/*` | Capability instances |
| `/task` | Process management |
| `/task/new` | Task creation |
| `/task/<id>/*` | Task control/status |

### Web Services

| Path | Description |
|------|-------------|
| `/web/dom` | DOM manipulation |
| `/web/dom/new/*` | Element creation |
| `/web/dom/<id>/*` | Element control |
| `/web/worker` | Web Workers |
| `/web/sw` | Service Worker |
| `/web/opfs` | Origin Private FS |
| `/web/ws` | WebSockets |

## JavaScript API

When running in browser:

```javascript
// Initialize Wanix
const wanix = await Wanix.init();

// File operations
await wanix.writeFile('/test.txt', 'Hello');
const data = await wanix.readFile('/test.txt');

// Directory operations
const entries = await wanix.readdir('/');

// Execute WASI program
const result = await wanix.exec('/bin/program', ['arg1', 'arg2']);
```

## Error Codes

Standard filesystem errors:

| Error | Description |
|-------|-------------|
| `ENOENT` | No such file or directory |
| `EEXIST` | File exists |
| `ENOTDIR` | Not a directory |
| `EISDIR` | Is a directory |
| `EACCES` | Permission denied |
| `EINVAL` | Invalid argument |
| `EIO` | I/O error |

## Conventions

### Path Conventions

- Absolute paths start with `/`
- No trailing slashes on directories
- `.` refers to current directory
- `..` moves up one directory

### Control File Format

```
COMMAND [ARGS...]
```

- Commands are single words
- Arguments are space-separated
- Quotes for arguments with spaces
- One command per write

### Response Format

- Success: Command-specific or empty
- Error: Error message string
- Async: May return immediately

## Version Compatibility

Current API version: **0.3**

- Breaking changes documented in release notes
- Backward compatibility maintained when possible
- Deprecated features marked clearly

## Getting Help

- Check service-specific documentation
- Use `help` command in control files
- Browse source code for examples
- Ask in Discord community