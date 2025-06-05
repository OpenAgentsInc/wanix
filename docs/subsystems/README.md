# Wanix Subsystems

This section provides detailed documentation for each major subsystem within Wanix. Each subsystem implements specific functionality while adhering to the "everything is a file" philosophy.

## Overview

Wanix subsystems are organized into logical groups:

1. **Core Systems** - Essential for basic operation
2. **Storage Systems** - Various filesystem implementations  
3. **Web Systems** - Browser platform integration
4. **Execution Systems** - Process and runtime support

## Subsystem Documentation

### Core Systems

#### [Filesystem](./filesystem.md)
The filesystem subsystem provides multiple implementations and abstractions:
- Memory filesystems (memfs, mapfs)
- Union filesystems for directory merging
- TAR archive support
- FUSE integration for native platforms
- Plan 9 protocol (9P) support

#### [Capabilities](./capabilities.md)
The capability system manages resource allocation and access control:
- Capability lifecycle management
- Resource allocation patterns (clone dance)
- Built-in capabilities (tarfs, tmpfs, loopback)
- Security enforcement

### Web Platform Systems

#### [Web Integration](./web-integration.md)
Browser-specific features exposed as file services:
- DOM manipulation through files
- Web Worker management
- Service Worker configuration
- File System Access API integration
- WebSocket connections

### Execution Systems

#### [WASI Runtime](./wasi-runtime.md)
WebAssembly System Interface support:
- WASI shim implementation
- Memory management
- File descriptor mapping
- Async I/O handling

#### [VM Subsystem](./vm-subsystem.md)
x86 emulation and Linux integration:
- v86 emulator integration
- Custom Linux kernel
- 9P filesystem bridge
- Device virtualization

## Design Principles

All subsystems follow these principles:

1. **File-based Interface**: All functionality exposed through filesystem operations
2. **Composability**: Subsystems can be combined to create complex behaviors
3. **Isolation**: Clear boundaries between subsystems
4. **Simplicity**: Each subsystem does one thing well
5. **Uniformity**: Consistent patterns across all subsystems

## Subsystem Interactions

```
┌─────────────────────────────────────────────┐
│              User Applications              │
└───────────────────┬─────────────────────────┘
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
┌───────────────┐       ┌───────────────┐
│   WASI Tasks  │       │  Linux Tasks  │
└───────┬───────┘       └───────┬───────┘
        │                       │
        └───────────┬───────────┘
                    ▼
        ┌─────────────────────────┐
        │      VFS Layer          │
        └───────────┬─────────────┘
                    │
    ┌───────────────┼───────────────┐
    ▼               ▼               ▼
┌────────┐    ┌────────┐    ┌────────┐
│  FS    │    │  Cap   │    │  Web   │
│Subsys  │    │Subsys  │    │Subsys  │
└────────┘    └────────┘    └────────┘
```

## Implementation Notes

### Language Choices
- **Core Systems**: Go for simplicity and built-in concurrency
- **WASI Shim**: TypeScript for browser integration
- **Linux Kernel**: C with minimal configuration
- **Build System**: Make with Docker for reproducibility

### Performance Considerations
- Subsystems are designed to minimize overhead
- File operations are optimized for common patterns
- Caching is used where appropriate
- Lazy initialization reduces startup time

### Security Model
- Subsystems enforce capability-based security
- No ambient authority between subsystems
- Clear trust boundaries
- Minimal attack surface

## Future Directions

- **Plugin Architecture**: Dynamic subsystem loading
- **Network Transparency**: Subsystems across network boundaries
- **Performance Monitoring**: Built-in profiling support
- **Extended Platform Support**: More native integrations