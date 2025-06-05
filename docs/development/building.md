# Building Wanix

This guide covers building Wanix from source, including all dependencies and build options.

## Prerequisites

### Required Tools

1. **Docker** (20.10.0+)
   - Used to build external dependencies
   - Ensures reproducible builds
   - [Install Docker](https://docs.docker.com/get-docker/)

2. **Go** (1.23+)
   - Primary development language
   - [Install Go](https://golang.org/dl/)

3. **TinyGo** (0.35+) - Optional but recommended
   - Produces smaller WASM binaries
   - [Install TinyGo](https://tinygo.org/getting-started/install/)

### System Requirements

- **Disk Space**: ~10GB for full build with dependencies
- **RAM**: 4GB minimum, 8GB recommended
- **OS**: macOS, Linux, or Windows with WSL2

## Build Process

### 1. Clone Repository

```bash
git clone https://github.com/tractordev/wanix.git
cd wanix
```

### 2. Build Dependencies

First time only - builds all external dependencies:

```bash
make deps
```

This builds:
- **v86**: x86 emulator (WebAssembly)
- **Linux kernel**: Custom minimal kernel
- **WASI shim**: WebAssembly System Interface
- **Shell environment**: BusyBox and utilities
- **VS Code assets**: For web-based IDE

**Note**: This step takes 10-30 minutes and requires ~10GB disk space.

### 3. Build Wanix

Standard build (uses TinyGo for WASM):

```bash
make build
```

This creates:
- `./wanix` - Native executable
- `wasm/assets/wanix.wasm` - WebAssembly module

## Build Options

### Using Go Instead of TinyGo

For faster builds with better debugging (but larger output):

```bash
make wasm-go wanix
```

Comparison:
- **TinyGo**: ~3MB WASM, slower build, limited stack traces
- **Go**: ~15MB WASM, faster build, full debugging

### Docker Build

If you have issues with native toolchain:

```bash
make docker
```

**Note**: Docker build doesn't include `console` command.

### Development Build

For rapid iteration during development:

```bash
# Skip WASM build if only working on native
go build ./cmd/wanix

# Skip native build if only working on WASM
make wasm
```

### Cross-Platform Builds

```bash
# Linux
GOOS=linux GOARCH=amd64 go build ./cmd/wanix

# Windows
GOOS=windows GOARCH=amd64 go build ./cmd/wanix

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build ./cmd/wanix

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build ./cmd/wanix
```

## Makefile Targets

### Primary Targets

| Target | Description |
|--------|-------------|
| `all` | Build everything (deps + build) |
| `deps` | Build external dependencies |
| `build` | Build Wanix binary and WASM |
| `clean` | Clean build artifacts |
| `clobber` | Clean everything (including deps) |

### Individual Dependencies

| Target | Description |
|--------|-------------|
| `dep-esbuild` | JavaScript bundler |
| `dep-linux` | Linux kernel |
| `dep-shell` | Shell environment |
| `dep-v86` | x86 emulator |
| `dep-vscode` | VS Code assets |
| `dep-wasi` | WASI shim |

### Utility Targets

| Target | Description |
|--------|-------------|
| `wasm` | Build only WASM module |
| `wasm-go` | Build WASM with Go |
| `wanix` | Build only native binary |
| `docker` | Full Docker build |

## Build Configuration

### Environment Variables

```bash
# Use specific Go version
GO=/usr/local/go1.23/bin/go make build

# Use specific TinyGo version
TINYGO=/opt/tinygo/bin/tinygo make build

# Skip TinyGo detection
WANIX_USE_GO=1 make build
```

### Build Tags

```bash
# Build without console support
go build -tags noconsole ./cmd/wanix

# Build with additional debugging
go build -tags debug ./cmd/wanix
```

## Dependency Details

### v86 Emulator

Built from [copy/v86](https://github.com/copy/v86):

```dockerfile
# external/v86/Dockerfile
FROM node:18 as builder
RUN git clone https://github.com/copy/v86.git
WORKDIR /v86
RUN make all
```

Patches applied:
- `patch/9p.js` - 9P filesystem fixes
- `patch/starter.js` - Wanix integration

### Linux Kernel

Custom minimal kernel configuration:

```bash
# Key features enabled
CONFIG_9P_FS=y          # 9P filesystem
CONFIG_VIRTIO=y         # Virtio drivers
CONFIG_PRINTK=n         # Disable kernel messages
CONFIG_MODULES=n        # No module support
```

### WASI Shim

TypeScript implementation compiled with esbuild:

```bash
# external/wasi/
npm install
npm run build

# Produces:
# - wasi.js (main shim)
# - worker.js (async I/O worker)
```

### Shell Environment

BusyBox-based with custom utilities:

```dockerfile
# external/shell/Dockerfile
FROM alpine:latest
RUN apk add --no-cache busybox-static

# Add Wanix-specific scripts
COPY bin/* /bin/
COPY etc/* /etc/
```

## Troubleshooting

### Common Issues

**1. Docker build fails**
```bash
# Check Docker daemon
docker version

# Clean Docker cache
docker system prune -a

# Increase Docker resources
# Docker Desktop > Preferences > Resources
```

**2. Out of disk space**
```bash
# Check space
df -h

# Clean Wanix build
make clean

# Clean dependencies
make clobber
```

**3. TinyGo not found**
```bash
# Install TinyGo
brew install tinygo  # macOS

# Or use Go instead
make wasm-go wanix
```

**4. Build hangs**
```bash
# Check for sufficient RAM
free -h

# Build with verbose output
make V=1 build
```

### Platform-Specific Issues

**macOS**
```bash
# If console doesn't work
# Install required libraries
brew install pkg-config
```

**Windows (WSL2)**
```bash
# Ensure WSL2 (not WSL1)
wsl --set-default-version 2

# Use Linux build commands
```

**Linux**
```bash
# May need additional packages
sudo apt-get install build-essential
```

## Build Optimization

### Faster Builds

1. **Parallel builds**
```bash
make -j$(nproc) deps
```

2. **Skip unnecessary targets**
```bash
# If not using console
make wasm wanix-noconsole
```

3. **Use build cache**
```bash
# Go module cache
export GOCACHE=$HOME/.cache/go-build
```

### Smaller Binaries

1. **Strip debug info**
```bash
go build -ldflags="-s -w" ./cmd/wanix
```

2. **Use TinyGo for WASM**
```bash
make wasm  # Uses TinyGo by default
```

3. **Compress assets**
```bash
# WASM is gzipped automatically by server
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Build
on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - uses: actions/setup-go@v4
      with:
        go-version: '1.23'
    
    - uses: acifani/setup-tinygo@v2
      with:
        tinygo-version: '0.35.0'
    
    - name: Build dependencies
      run: make deps
    
    - name: Build Wanix
      run: make build
    
    - name: Run tests
      run: go test ./...
```

## Next Steps

After building successfully:

1. **Run Wanix**: `./wanix serve`
2. **Access in browser**: http://localhost:8080
3. **Read development guide**: [Creating File Services](./creating-file-services.md)
4. **Run tests**: [Testing Guide](./testing.md)