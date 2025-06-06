# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Build everything (dependencies + Wanix)
make all

# Build only dependencies (required first time)
make deps

# Build Wanix binary and WASM module
make build

# Build WASM with Go instead of TinyGo (larger but faster)
make wasm-go wanix

# Build using Docker (without console command)
make docker

# Clean all build artifacts
make clobber
```

## Development Commands

```bash
# Run tests
go test ./...

# Run Wanix web server
./wanix serve

# Run console (if built with console support)
./wanix console
```

## Architecture Overview

Wanix is a capability-oriented microkernel implementing Plan 9's "everything is a file" philosophy. The core kernel (`wanix.go`) consists of:

- **Cap Service**: Manages capabilities for resource access control
- **Task Service**: Handles process/task management with per-process namespaces
- **VFS**: Virtual filesystem providing namespace abstraction
- **Module System**: Pluggable modules that bind into the namespace

Key architectural patterns:
- All resources are exposed as files through VFS
- Processes have isolated namespaces for security
- Capabilities control access to resources
- Web platform features (DOM, workers, OPFS) are exposed as file interfaces

## Core Subsystems

**Filesystem (`/fs/`)**: Multiple filesystem implementations including:
- `fskit/`: Core abstractions (memfs, mapfs, unionfs)
- `p9kit/`: Plan 9 protocol implementation
- `fusekit/`: FUSE support for native platforms
- `tarfs/`: TAR archive filesystem

**Web Integration (`/web/`)**: Browser platform integrations:
- `dom/`: DOM manipulation through file interfaces
- `worker/`: Web Worker management
- `sw/`: Service Worker support
- `fsa/`: File System Access API
- `vm/`: x86 emulation support

**Process Model (`/task/`)**: Abstract POSIX-like process management supporting both WASI WebAssembly and x86 executables.

## Testing Strategy

Use Go's standard testing framework. Test files follow `*_test.go` convention throughout the codebase.

Run all tests: `go test ./...`

Key test locations:
- `/fs/fskit/*_test.go`: Filesystem implementations
- `/fs/p9kit/*_test.go`: Plan 9 protocol
- `/vfs/*_test.go`: Virtual filesystem
- `/cap/*_test.go`: Capability system

## Build System Notes

The build uses Make with Docker for reproducible dependency builds. External dependencies (v86, Linux kernel, WASI shim, shell environment) are built in Docker containers and embedded into the binary.

Prerequisites:
- Docker 20.10+ (for dependency builds)
- Go 1.23+ (for native builds)
- TinyGo 0.35+ (optional, for smaller WASM)

The `make deps` step builds all external dependencies and only needs to run once. Subsequent builds use `make build`.

## Important Instructions

- NEVER start the Wanix server with `./wanix serve` - the user must do this themselves to avoid tying up the thread
- Always let the user control server lifecycle
- ALWAYS update the server version string in `cmd/wanix/serve.go` when making changes so the user knows they're running the latest version (e.g., "Server v21:00 - Added debug logging")
