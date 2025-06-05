# Development Guide

This guide provides everything you need to develop with and contribute to Wanix.

## Overview

Whether you're building applications for Wanix, extending its capabilities, or contributing to the core project, this guide will help you get started.

## Development Documentation

### [Building Wanix](./building.md)
Complete build instructions:
- Prerequisites and setup
- Building from source
- Docker-based builds
- Troubleshooting common issues

### [Creating File Services](./creating-file-services.md)
Extend Wanix with new capabilities:
- File service architecture
- Implementation patterns
- Integration with the kernel
- Best practices and examples

### [Testing](./testing.md)
Testing strategies and practices:
- Unit testing approach
- Integration testing
- Browser testing
- Performance testing

### [Debugging](./debugging.md)
Tools and techniques for debugging:
- Browser DevTools integration
- Logging and tracing
- Common issues and solutions
- Performance profiling

## Quick Start

### Setting Up Development Environment

1. **Clone the repository**
```bash
git clone https://github.com/tractordev/wanix.git
cd wanix
```

2. **Install prerequisites**
- Docker 20.10+
- Go 1.23+
- TinyGo 0.35+ (optional)

3. **Build dependencies**
```bash
make deps
```

4. **Build Wanix**
```bash
make build
```

5. **Run development server**
```bash
./wanix serve
```

## Development Workflow

### Making Changes

1. **Create a feature branch**
```bash
git checkout -b feature/my-feature
```

2. **Make your changes**
- Follow existing code patterns
- Add tests for new functionality
- Update documentation

3. **Test your changes**
```bash
# Run tests
go test ./...

# Test in browser
./wanix serve
# Navigate to http://localhost:8080
```

4. **Submit a pull request**
- Clear description of changes
- Link to relevant issues
- Ensure CI passes

### Code Organization

```
wanix/
├── cmd/           # CLI commands
├── fs/            # Filesystem implementations
├── cap/           # Capability system
├── task/          # Process management
├── web/           # Web platform integration
├── vfs/           # Virtual filesystem
├── external/      # External dependencies
└── wasm/          # WebAssembly module
```

## Development Principles

### 1. Everything is a File
- Expose all functionality through filesystem interfaces
- Use standard file operations (read, write, ctl)
- Follow Plan 9 conventions where applicable

### 2. Simplicity First
- Prefer simple, composable solutions
- Avoid unnecessary abstractions
- Clear is better than clever

### 3. Security by Design
- Capability-based security model
- No ambient authority
- Principle of least privilege

### 4. Web-First, Platform-Agnostic
- Design for browser environment first
- Maintain compatibility with native platforms
- Progressive enhancement

## Common Development Tasks

### Adding a New File Service

1. Create service structure in appropriate package
2. Implement `fs.FS` interface
3. Register with module system
4. Add tests
5. Document control commands

Example:
```go
type MyService struct {
    data map[string]string
}

func (s *MyService) Open(name string) (fs.File, error) {
    // Implementation
}
```

### Adding a New Capability

1. Define capability interface
2. Implement allocation logic
3. Register with capability service
4. Add documentation

Example:
```go
func init() {
    cap.Register("mytype", &MyCapabilityType{})
}
```

### Debugging WebAssembly

1. Use browser DevTools
2. Enable source maps in build
3. Add console logging
4. Use WASM debugging extensions

### Performance Optimization

1. Profile with browser tools
2. Minimize allocations
3. Use efficient data structures
4. Cache when appropriate

## Getting Help

### Resources

- [GitHub Issues](https://github.com/tractordev/wanix/issues)
- [Discord Community](https://discord.gg/nQbgRjEBU4)
- [Architecture Docs](../architecture/README.md)
- [API Reference](../api-reference/README.md)

### Common Issues

**Build fails with "out of space"**
- Docker builds require significant disk space
- Clean up with `docker system prune`

**WASM module too large**
- Ensure using TinyGo for production builds
- Check for unnecessary dependencies

**Browser compatibility issues**
- Check browser console for errors
- Ensure modern browser with WASM support
- Some features (FSA) have limited support

## Contributing

We welcome contributions! Please:

1. Read [CONTRIBUTING.md](../../CONTRIBUTING.md)
2. Check existing issues and discussions
3. Follow code style and conventions
4. Add tests for new features
5. Update documentation

### Areas for Contribution

- **New file services**: Extend platform capabilities
- **Performance**: Optimization and profiling
- **Documentation**: Improve guides and examples
- **Testing**: Increase test coverage
- **Platform support**: Native platform features