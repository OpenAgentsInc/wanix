# Wanix Documentation

Welcome to the Wanix documentation. This guide provides comprehensive information about the Wanix virtual environment toolchain, its architecture, and development practices.

## What is Wanix?

Wanix is a capability-oriented microkernel that brings Plan 9's "everything is a file" philosophy to the web. It allows you to build virtual environments that run in browsers and natively, supporting both WebAssembly (WASI) and x86 executables.

## 🚀 [Quick Reference](./quick-reference.md)
**New to Wanix?** Start here for a concise overview of key concepts and common tasks.

## Documentation Structure

### 📚 [Architecture](./architecture/README.md)
Deep dive into Wanix's design principles and core systems:
- [Core Concepts](./architecture/core-concepts.md) - Namespaces, capabilities, and file services
- [VFS Design](./architecture/vfs-design.md) - Virtual filesystem architecture
- [Process Model](./architecture/process-model.md) - Task management and isolation
- [Module System](./architecture/module-system.md) - How modules extend Wanix

### 🔧 [Subsystems](./subsystems/README.md)
Detailed documentation of major components:
- [Filesystem](./subsystems/filesystem.md) - File system implementations
- [Capabilities](./subsystems/capabilities.md) - Capability management system
- [Web Integration](./subsystems/web-integration.md) - Browser platform features
- [WASI Runtime](./subsystems/wasi-runtime.md) - WebAssembly support
- [VM Subsystem](./subsystems/vm-subsystem.md) - x86 emulation and Linux integration

### 👨‍💻 [Development Guide](./development/README.md)
Everything you need to develop with and for Wanix:
- [Building Wanix](./development/building.md) - Build process and dependencies
- [Creating File Services](./development/creating-file-services.md) - Extend Wanix with new capabilities
- [Testing](./development/testing.md) - Testing strategies and practices
- [Debugging](./development/debugging.md) - Tips for debugging Wanix

### 📖 [API Reference](./api-reference/README.md)
Complete API documentation:
- [VFS API](./api-reference/vfs-api.md) - Core filesystem operations
- [File Services](./api-reference/file-services.md) - Built-in services reference
- [Control Files](./api-reference/control-files.md) - Control file command reference

### 📹 [Transcripts](./transcripts/)
Video and presentation transcripts for additional context

## Quick Start

1. **Install Wanix**: See [Building Wanix](./development/building.md)
2. **Understand the basics**: Read [Core Concepts](./architecture/core-concepts.md)
3. **Try it online**: Visit [wanix.run](https://wanix.run)

## Key Concepts Summary

- **Everything is a File**: All system resources are exposed through the filesystem
- **Per-Process Namespaces**: Each process has its own view of the filesystem
- **Capabilities through File Services**: Access to resources is controlled by what's in your namespace
- **Multi-Runtime Support**: Run both WASI WebAssembly and x86 Linux executables

## Getting Help

- [GitHub Issues](https://github.com/tractordev/wanix/issues)
- [Discord Community](https://discord.gg/nQbgRjEBU4)
- [GitHub Discussions](https://github.com/tractordev/wanix/discussions)