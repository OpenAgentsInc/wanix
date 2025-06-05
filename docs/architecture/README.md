# Wanix Architecture

This section provides a comprehensive overview of Wanix's architecture, design principles, and core systems.

## Overview

Wanix is built on several key architectural principles inspired by Plan 9:

1. **Everything is a File** - All system resources are exposed as files
2. **Per-Process Namespaces** - Each process has its own filesystem view
3. **Capability-Oriented Design** - Access control through namespace composition
4. **Composable File Services** - Build complex systems from simple primitives

## Architecture Documents

### [Core Concepts](./core-concepts.md)
Fundamental concepts that drive Wanix's design:
- Namespaces and their role in isolation
- Capabilities and security model
- File services as system primitives
- The "clone dance" pattern

### [VFS Design](./vfs-design.md)
The Virtual Filesystem at the heart of Wanix:
- VFS architecture and implementation
- Filesystem routing and delegation
- Union mounts and binding
- Device files and synthetic filesystems

### [Process Model](./process-model.md)
How Wanix manages processes and tasks:
- Task creation and lifecycle
- Namespace inheritance and modification
- Environment isolation
- Multi-runtime support (WASI and x86)

### [Module System](./module-system.md)
Extensibility through modules:
- Module loading and initialization
- Service registration
- Namespace integration
- Built-in modules overview

## System Layers

```
┌─────────────────────────────────────────┐
│         User Applications               │
│    (WASI binaries, x86 programs)       │
├─────────────────────────────────────────┤
│         Shell & Utilities               │
│    (BusyBox, custom scripts)           │
├─────────────────────────────────────────┤
│        File Services Layer              │
│ (DOM, Workers, OPFS, VM, Capabilities) │
├─────────────────────────────────────────┤
│      Virtual Filesystem (VFS)           │
│   (Namespaces, unions, bindings)       │
├─────────────────────────────────────────┤
│         Core Services                   │
│    (Cap Service, Task Service)         │
├─────────────────────────────────────────┤
│         Wanix Kernel                    │
│    (Module loader, initializer)        │
└─────────────────────────────────────────┘
```

## Key Design Decisions

### Why Plan 9?
Plan 9's design offers unparalleled composability and simplicity. By treating everything as files and giving each process its own namespace, we achieve:
- Natural sandboxing and security
- Easy resource virtualization
- Simple, uniform APIs
- Powerful composition primitives

### Web-First Architecture
Wanix is designed to run efficiently in browsers while maintaining compatibility with native platforms:
- WebAssembly as primary execution target
- Browser APIs exposed as file services
- Service Worker integration for offline support
- Progressive enhancement for native features

### Capability Security
Instead of traditional permission systems, Wanix uses capabilities:
- If it's not in your namespace, you can't access it
- File services act as capability tokens
- Fine-grained access control through namespace composition
- No ambient authority

## Implementation Languages

- **Core Kernel**: Go (compiles to both native and WASM)
- **External Dependencies**: C/C++ (compiled in Docker)
- **Web Integration**: TypeScript/JavaScript
- **Shell Environment**: POSIX shell scripts