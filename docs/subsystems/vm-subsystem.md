# VM Subsystem

The VM subsystem enables Wanix to run x86 Linux executables through the v86 emulator, providing compatibility with existing Linux software while maintaining Wanix's filesystem-based architecture.

## Overview

The VM subsystem provides:
- **x86 emulation** via v86 (WebAssembly JIT)
- **Custom Linux kernel** with 9P root filesystem
- **Seamless integration** with Wanix namespace
- **BusyBox utilities** for shell environment
- **Device virtualization** for I/O

## Architecture

```
┌─────────────────────────────────────┐
│      Linux Application (x86)        │
│         (BusyBox, etc.)            │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│         Linux Kernel                │
│    (Custom minimal build)           │
└──────────────┬──────────────────────┘
               │ 9P Protocol
┌──────────────▼──────────────────────┐
│         9P Server                   │
│    (Bridges to Wanix VFS)          │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│       v86 Emulator (WASM)          │
│    (x86 to WebAssembly JIT)        │
└─────────────────────────────────────┘
```

## v86 Integration (`web/vm/`)

### VM Service (`vm/service.go`)

```go
type Service struct {
    vms map[string]*VM
    mu  sync.RWMutex
}

type VM struct {
    id       string
    v86      js.Value
    kernel   []byte
    initrd   []byte
    p9server *p9kit.Server
    console  io.ReadWriter
    status   string
}

func (s *Service) CreateVM() (*VM, error) {
    vm := &VM{
        id:     generateID(),
        kernel: loadKernel(),
        initrd: loadInitrd(),
    }
    
    // Initialize v86
    vm.v86 = js.Global().Get("V86").New(map[string]interface{}{
        "wasm_path": "/v86.wasm",
        "memory_size": 128 * 1024 * 1024, // 128MB
        "vga_memory_size": 2 * 1024 * 1024,
        "autostart": false,
    })
    
    return vm, nil
}
```

### v86 Configuration (`vm/v86.go`)

```javascript
// v86 emulator setup
const emulator = new V86({
    wasm_path: "/v86.wasm",
    bios: { url: "/seabios.bin" },
    kernel: {
        url: "/vmlinux",
        cmdline: "console=ttyS0 root=/dev/root rootfstype=9p rootflags=trans=virtio,cache=loose"
    },
    filesystem: {
        baseurl: "/",
        basefs: "9p"
    },
    autostart: true,
    disable_keyboard: true,
    disable_mouse: true,
    
    // Network configuration
    network_adapter: {
        type: "virtio",
        relay_url: "ws://localhost:8080/"
    },
    
    // Serial console
    serial_container_xtermjs: terminalElement
});
```

## Custom Linux Kernel (`external/linux/`)

### Kernel Configuration (`kernel.config`)

Minimal kernel configuration for Wanix:

```
# Core options
CONFIG_EXPERT=y
CONFIG_EMBEDDED=y
CONFIG_OPTIMIZE_FOR_SIZE=y

# Filesystem support
CONFIG_9P_FS=y
CONFIG_9P_FS_POSIX_ACL=y
CONFIG_NET_9P=y
CONFIG_NET_9P_VIRTIO=y

# Virtio drivers
CONFIG_VIRTIO=y
CONFIG_VIRTIO_PCI=y
CONFIG_VIRTIO_CONSOLE=y
CONFIG_VIRTIO_NET=y

# Disable unnecessary features
# CONFIG_MODULES is not set
# CONFIG_BLOCK is not set
# CONFIG_NETWORK_FILESYSTEMS is not set
```

### Build Process (`linux/Makefile`)

```makefile
# Build minimal Linux kernel
kernel: .config
	make -j$(nproc) ARCH=i386 bzImage
	cp arch/x86/boot/bzImage /output/vmlinux

# Strip unnecessary sections
	strip --strip-unneeded /output/vmlinux
	
# Compress with custom loader
	cat boot-stub /output/vmlinux | gzip > /output/vmlinux.gz
```

## 9P Server Bridge (`fs/p9kit/`)

### Server Implementation

```go
type Server struct {
    fs   fs.FS
    fids map[uint32]*Fid
    mu   sync.RWMutex
}

func (s *Server) Handle(msg Message) (Response, error) {
    switch msg.Type {
    case Tversion:
        return s.handleVersion(msg)
    case Tattach:
        return s.handleAttach(msg)
    case Twalk:
        return s.handleWalk(msg)
    case Topen:
        return s.handleOpen(msg)
    case Tread:
        return s.handleRead(msg)
    case Twrite:
        return s.handleWrite(msg)
    // ... other 9P operations
    }
}
```

### Filesystem Mapping

```go
// Map Wanix VFS to 9P
func (s *Server) handleWalk(msg *Twalk) (*Rwalk, error) {
    fid := s.getFid(msg.Fid)
    if fid == nil {
        return nil, ErrBadFid
    }
    
    // Walk path components
    qids := make([]Qid, 0, len(msg.Names))
    current := fid.Path
    
    for _, name := range msg.Names {
        next := path.Join(current, name)
        
        // Check existence in Wanix VFS
        info, err := s.fs.Stat(next)
        if err != nil {
            break
        }
        
        qids = append(qids, statToQid(info))
        current = next
    }
    
    // Create new fid
    if msg.NewFid != msg.Fid {
        s.putFid(msg.NewFid, &Fid{Path: current})
    }
    
    return &Rwalk{Qids: qids}, nil
}
```

## VM Lifecycle Management

### Starting a VM

```go
func (vm *VM) Start(ns *vfs.Namespace) error {
    // Create 9P server for namespace
    vm.p9server = p9kit.NewServer(ns)
    
    // Set up virtio-9p transport
    vm.v86.Call("set_virtio_9p_filesystem", map[string]interface{}{
        "mount_tag": "root",
        "server": vm.p9server,
    })
    
    // Configure serial console
    vm.v86.Set("serial0_send", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        char := byte(args[0].Int())
        vm.console.Write([]byte{char})
        return nil
    }))
    
    // Start emulation
    vm.v86.Call("run")
    vm.status = "running"
    
    return nil
}
```

### Console I/O

```go
type ConsoleDevice struct {
    vm     *VM
    input  chan byte
    output chan byte
}

func (c *ConsoleDevice) Write(p []byte) (int, error) {
    // Send to VM
    for _, b := range p {
        c.vm.v86.Call("serial0_send_char", b)
    }
    return len(p), nil
}

func (c *ConsoleDevice) Read(p []byte) (int, error) {
    // Read from VM output buffer
    n := 0
    for n < len(p) {
        select {
        case b := <-c.output:
            p[n] = b
            n++
        default:
            if n > 0 {
                return n, nil
            }
            // Block for at least one byte
            p[0] = <-c.output
            return 1, nil
        }
    }
    return n, nil
}
```

## Shell Environment (`shell/`)

### BusyBox Integration

The shell environment provides Unix utilities:

```dockerfile
# shell/Dockerfile
FROM alpine:latest as builder

# Build static BusyBox
RUN apk add --no-cache gcc musl-dev make
RUN wget https://busybox.net/downloads/busybox-${VERSION}.tar.bz2
RUN tar xf busybox-${VERSION}.tar.bz2

WORKDIR /busybox-${VERSION}
RUN make defconfig
RUN make CONFIG_STATIC=y

# Create minimal root filesystem
FROM scratch
COPY --from=builder /busybox-${VERSION}/busybox /bin/busybox

# Create symlinks for all applets
RUN /bin/busybox --install -s /bin/
```

### Wanix Shell Scripts

Custom utilities for Wanix integration:

```bash
#!/bin/sh
# domctl - DOM control utility

case "$1" in
    new)
        type="$2"
        if [ -z "$type" ]; then
            echo "Usage: domctl new <element-type>"
            echo "Available types:"
            ls /web/dom/new/
            exit 1
        fi
        cat "/web/dom/new/$type"
        ;;
    
    body)
        shift
        echo "$@" > /web/dom/body/ctl
        ;;
    
    *)
        id="$1"
        cmd="$2"
        shift 2
        echo "$cmd $@" > "/web/dom/$id/ctl"
        ;;
esac
```

## Device Emulation

### Serial Console

```javascript
// Virtio console device
class VirtioConsole {
    constructor(emulator) {
        this.emulator = emulator;
        this.input_buffer = [];
        this.output_buffer = [];
    }
    
    send_char(char) {
        this.input_buffer.push(char);
        this.emulator.trigger_irq(VIRTIO_CONSOLE_IRQ);
    }
    
    read_char() {
        return this.output_buffer.shift() || -1;
    }
}
```

### Network Device

```javascript
// Virtio network adapter
class VirtioNet {
    constructor(emulator, relay_url) {
        this.emulator = emulator;
        this.socket = new WebSocket(relay_url);
        this.setupHandlers();
    }
    
    send_packet(data) {
        if (this.socket.readyState === WebSocket.OPEN) {
            this.socket.send(data);
        }
    }
    
    receive_packet(data) {
        // Queue packet for VM
        this.rx_queue.push(data);
        this.emulator.trigger_irq(VIRTIO_NET_IRQ);
    }
}
```

## Performance Optimization

### JIT Compilation

v86 uses dynamic recompilation for performance:

```javascript
// Hot path detection and compilation
class JITCompiler {
    compile_basic_block(start_addr) {
        const instructions = this.decode_block(start_addr);
        const wasm_code = this.generate_wasm(instructions);
        
        // Compile to WebAssembly
        const module = new WebAssembly.Module(wasm_code);
        const instance = new WebAssembly.Instance(module, {
            env: this.emulator.exports
        });
        
        // Cache compiled block
        this.cache.set(start_addr, instance.exports.execute);
    }
}
```

### Memory Management

```go
// Efficient memory sharing
type SharedMemory struct {
    buffer *js.Value // SharedArrayBuffer
    view   *js.Value // DataView
}

func (m *SharedMemory) Read(offset, length int) []byte {
    data := make([]byte, length)
    js.CopyBytesToGo(data, m.buffer.Call("slice", offset, offset+length))
    return data
}
```

## Integration Examples

### Running Linux Commands

```go
// Execute command in VM
func (vm *VM) Exec(cmd string) (string, error) {
    // Send command to shell
    vm.console.Write([]byte(cmd + "\n"))
    
    // Read output until prompt
    output := &bytes.Buffer{}
    prompt := []byte("# ")
    
    for {
        buf := make([]byte, 1024)
        n, _ := vm.console.Read(buf)
        output.Write(buf[:n])
        
        if bytes.HasSuffix(output.Bytes(), prompt) {
            break
        }
    }
    
    return output.String(), nil
}
```

### File Sharing

```bash
# In VM, access Wanix files
ls /              # Shows 9P root
cat /web/opfs/data.txt

# Write creates files in Wanix
echo "Hello from Linux" > /tmp/output.txt
```

## Security Considerations

### VM Isolation

```go
// Restricted namespace for VM
func createVMNamespace(parent *vfs.Namespace) *vfs.Namespace {
    ns := vfs.NewNamespace()
    
    // Only expose safe directories
    ns.Bind(parent, "/tmp", "/tmp")
    ns.Bind(parent, "/web/opfs", "/data")
    
    // No access to capabilities or system
    // No access to other processes
    
    return ns
}
```

### Resource Limits

```javascript
// Enforce VM resource limits
const vm_config = {
    memory_size: 128 * 1024 * 1024,  // 128MB max
    cpu_throttle: 0.5,                // 50% CPU max
    disk_quota: 1024 * 1024 * 1024,   // 1GB disk
    network_rate_limit: 1024 * 1024   // 1MB/s
};
```

## Debugging Support

### VM State Inspection

```go
func (vm *VM) GetRegisters() map[string]uint32 {
    regs := make(map[string]uint32)
    
    state := vm.v86.Call("get_cpu_state")
    regs["eax"] = uint32(state.Get("eax").Int())
    regs["ebx"] = uint32(state.Get("ebx").Int())
    // ... other registers
    
    return regs
}
```

### Trace Logging

```javascript
// Enable instruction tracing
emulator.log_instructions = true;
emulator.log_level = LOG_ALL;

emulator.add_listener("instruction", (addr, opcode) => {
    console.log(`[VM] ${addr.toString(16)}: ${opcode.toString(16)}`);
});
```

## Future Enhancements

- **KVM Support**: Hardware acceleration when available
- **Multiple Architectures**: ARM, RISC-V support
- **Container Integration**: OCI container runtime
- **GPU Passthrough**: WebGPU acceleration
- **Persistent VMs**: Save/restore VM state