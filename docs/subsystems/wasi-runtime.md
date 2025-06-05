# WASI Runtime Subsystem

The WASI (WebAssembly System Interface) runtime enables Wanix to execute WebAssembly modules compiled for WASI, providing a bridge between WASM programs and the Wanix filesystem.

## Overview

The WASI runtime provides:
- **WASI Preview 1** implementation
- **Filesystem access** through Wanix VFS
- **Process management** integration
- **Memory isolation** between modules
- **Async I/O support** in browser environment

## Architecture

```
┌─────────────────────────────────────┐
│         WASI Application            │
│        (Rust/Go/Zig/C)             │
└──────────────┬──────────────────────┘
               │ WASI Calls
┌──────────────▼──────────────────────┐
│          WASI Shim                  │
│    (TypeScript/JavaScript)          │
└──────────────┬──────────────────────┘
               │ VFS Operations
┌──────────────▼──────────────────────┐
│         Wanix VFS                   │
│    (Namespace/File Access)          │
└─────────────────────────────────────┘
```

## WASI Shim Implementation (`external/wasi/`)

### Core Structure

```typescript
// index.ts
export class WASI {
    private memory: WebAssembly.Memory;
    private view: DataView;
    private textDecoder: TextDecoder;
    private textEncoder: TextEncoder;
    private namespace: Namespace;
    private fds: FileDescriptor[];
    
    constructor(options: WASIOptions) {
        this.namespace = options.namespace;
        this.fds = this.initializeFileDescriptors(options);
    }
    
    // Import object for WebAssembly
    get imports() {
        return {
            wasi_snapshot_preview1: {
                fd_read: this.fd_read.bind(this),
                fd_write: this.fd_write.bind(this),
                fd_close: this.fd_close.bind(this),
                path_open: this.path_open.bind(this),
                // ... other WASI functions
            }
        };
    }
}
```

### File Descriptor Management

```typescript
// fs.ts
interface FileDescriptor {
    fd: number;
    path: string;
    flags: number;
    handle: FileHandle;
    position: bigint;
}

class FileDescriptorTable {
    private fds: Map<number, FileDescriptor> = new Map();
    private nextFd: number = 3; // 0,1,2 reserved
    
    open(path: string, flags: number): number {
        const fd = this.nextFd++;
        const handle = this.namespace.open(path, flags);
        
        this.fds.set(fd, {
            fd,
            path,
            flags,
            handle,
            position: 0n
        });
        
        return fd;
    }
    
    get(fd: number): FileDescriptor {
        const descriptor = this.fds.get(fd);
        if (!descriptor) {
            throw new Error(`Invalid file descriptor: ${fd}`);
        }
        return descriptor;
    }
}
```

### Memory Management

```typescript
// Memory view helpers
class MemoryView {
    constructor(private memory: WebAssembly.Memory) {}
    
    getString(ptr: number, len: number): string {
        const view = new Uint8Array(this.memory.buffer, ptr, len);
        return new TextDecoder().decode(view);
    }
    
    setString(ptr: number, str: string): number {
        const encoded = new TextEncoder().encode(str);
        const view = new Uint8Array(this.memory.buffer, ptr);
        view.set(encoded);
        return encoded.length;
    }
    
    readStruct<T>(ptr: number, struct: StructDef<T>): T {
        const view = new DataView(this.memory.buffer);
        return struct.read(view, ptr);
    }
}
```

## WASI System Calls

### File I/O Operations

```typescript
// fd_read implementation
fd_read(fd: number, iovs_ptr: number, iovs_len: number, nread_ptr: number): number {
    try {
        const descriptor = this.fds.get(fd);
        const iovs = this.readIovecs(iovs_ptr, iovs_len);
        
        let totalRead = 0;
        for (const iov of iovs) {
            const buffer = new Uint8Array(this.memory.buffer, iov.ptr, iov.len);
            const bytesRead = descriptor.handle.read(buffer);
            totalRead += bytesRead;
            
            if (bytesRead < iov.len) break;
        }
        
        this.view.setUint32(nread_ptr, totalRead, true);
        return 0; // Success
    } catch (e) {
        return this.errorToWasi(e);
    }
}

// fd_write implementation
fd_write(fd: number, iovs_ptr: number, iovs_len: number, nwritten_ptr: number): number {
    try {
        const descriptor = this.fds.get(fd);
        const iovs = this.readIovecs(iovs_ptr, iovs_len);
        
        let totalWritten = 0;
        for (const iov of iovs) {
            const buffer = new Uint8Array(this.memory.buffer, iov.ptr, iov.len);
            const bytesWritten = descriptor.handle.write(buffer);
            totalWritten += bytesWritten;
        }
        
        this.view.setUint32(nwritten_ptr, totalWritten, true);
        return 0;
    } catch (e) {
        return this.errorToWasi(e);
    }
}
```

### Path Operations

```typescript
// path_open implementation
path_open(
    dirfd: number,
    dirflags: number,
    path_ptr: number,
    path_len: number,
    oflags: number,
    fs_rights_base: bigint,
    fs_rights_inheriting: bigint,
    fdflags: number,
    fd_ptr: number
): number {
    try {
        const path = this.getString(path_ptr, path_len);
        const fullPath = this.resolvePath(dirfd, path);
        
        // Check permissions
        if (!this.checkRights(dirfd, fs_rights_base)) {
            return WASI_ERRNO_NOTCAPABLE;
        }
        
        // Open file
        const newFd = this.fds.open(fullPath, oflags);
        this.view.setUint32(fd_ptr, newFd, true);
        
        return 0;
    } catch (e) {
        return this.errorToWasi(e);
    }
}
```

### Directory Operations

```typescript
// fd_readdir implementation
fd_readdir(
    fd: number,
    buf_ptr: number,
    buf_len: number,
    cookie: bigint,
    bufused_ptr: number
): number {
    try {
        const descriptor = this.fds.get(fd);
        const entries = descriptor.handle.readdir();
        
        let offset = 0;
        let entryCount = Number(cookie);
        
        while (offset < buf_len && entryCount < entries.length) {
            const entry = entries[entryCount];
            const encoded = this.encodeDirent(entry);
            
            if (offset + encoded.length > buf_len) break;
            
            new Uint8Array(this.memory.buffer, buf_ptr + offset).set(encoded);
            offset += encoded.length;
            entryCount++;
        }
        
        this.view.setUint32(bufused_ptr, offset, true);
        return 0;
    } catch (e) {
        return this.errorToWasi(e);
    }
}
```

## Async I/O Support

### Worker-based Implementation (`worker.js`)

```javascript
// Synchronous worker for blocking I/O
self.onmessage = async (e) => {
    const { id, method, args } = e.data;
    
    try {
        const result = await callVFS(method, args);
        self.postMessage({ id, result });
    } catch (error) {
        self.postMessage({ id, error: error.message });
    }
};

async function callVFS(method, args) {
    switch (method) {
        case 'read':
            return await namespace.read(args.path, args.offset, args.length);
        case 'write':
            return await namespace.write(args.path, args.data, args.offset);
        // ... other operations
    }
}
```

### Synchronous Bridge (`worker_sync.js`)

```javascript
// SharedArrayBuffer-based synchronization
class SyncBridge {
    constructor(sab) {
        this.sab = sab;
        this.int32 = new Int32Array(sab);
        this.uint8 = new Uint8Array(sab, 8);
    }
    
    call(method, args) {
        // Write request
        this.writeRequest(method, args);
        
        // Signal worker
        Atomics.store(this.int32, 0, 1);
        Atomics.notify(this.int32, 0);
        
        // Wait for response
        Atomics.wait(this.int32, 0, 1);
        
        // Read response
        return this.readResponse();
    }
}
```

## Poll/Event Support (`poll-oneoff.ts`)

### Event Subscription

```typescript
interface Subscription {
    userdata: bigint;
    type: number;
    union: FdReadwrite | Clock;
}

function poll_oneoff(
    in_ptr: number,
    out_ptr: number,
    nsubscriptions: number,
    nevents_ptr: number
): number {
    const subscriptions = readSubscriptions(in_ptr, nsubscriptions);
    const events: Event[] = [];
    
    for (const sub of subscriptions) {
        switch (sub.type) {
            case EVENTTYPE_CLOCK:
                handleClockEvent(sub, events);
                break;
            case EVENTTYPE_FD_READ:
                handleFdReadEvent(sub, events);
                break;
            case EVENTTYPE_FD_WRITE:
                handleFdWriteEvent(sub, events);
                break;
        }
    }
    
    // Write events
    writeEvents(out_ptr, events);
    view.setUint32(nevents_ptr, events.length, true);
    
    return 0;
}
```

## Integration with Wanix

### Task Creation

```go
// Creating a WASI task
func createWASITask(wasmPath string, ns *vfs.Namespace) (*Task, error) {
    // Read WASM module
    wasmData, err := fs.ReadFile(ns, wasmPath)
    if err != nil {
        return nil, err
    }
    
    // Create task
    task := &Task{
        Type:      TaskTypeWASI,
        Namespace: ns,
        Module:    wasmData,
    }
    
    // Initialize WASI runtime
    task.wasi = wasi.New(wasi.Options{
        Namespace: ns,
        Args:      task.Args,
        Env:       task.Env,
        Stdin:     task.Files[0],
        Stdout:    task.Files[1],
        Stderr:    task.Files[2],
    })
    
    return task, nil
}
```

### Namespace Mapping

```typescript
// Map Wanix namespace to WASI
class WanixNamespace implements WASINamespace {
    constructor(private vfs: VFS) {}
    
    async open(path: string, flags: number): Promise<Handle> {
        const file = await this.vfs.open(path, flags);
        return new WanixHandle(file);
    }
    
    async stat(path: string): Promise<Filestat> {
        const info = await this.vfs.stat(path);
        return this.toWASIFilestat(info);
    }
}
```

## Supported Languages

### Rust
```rust
// Example WASI program
use std::fs;
use std::io::Write;

fn main() {
    // Read from Wanix filesystem
    let content = fs::read_to_string("/web/opfs/data.txt")
        .expect("Failed to read file");
    
    // Write to stdout
    println!("Content: {}", content);
    
    // Create new file
    let mut file = fs::File::create("/tmp/output.txt")
        .expect("Failed to create file");
    file.write_all(b"Hello from Rust!")
        .expect("Failed to write");
}
```

### Go
```go
//go:build wasi

package main

import (
    "fmt"
    "os"
)

func main() {
    // Access Wanix filesystem
    data, err := os.ReadFile("/web/opfs/config.json")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Config: %s\n", data)
    
    // Create file
    err = os.WriteFile("/tmp/result.txt", []byte("Go WASI"), 0644)
    if err != nil {
        panic(err)
    }
}
```

### Zig
```zig
const std = @import("std");

pub fn main() !void {
    // Open file from Wanix
    const file = try std.fs.openFileAbsolute("/web/opfs/data.txt", .{});
    defer file.close();
    
    // Read content
    const content = try file.readToEndAlloc(std.heap.page_allocator, 1024);
    defer std.heap.page_allocator.free(content);
    
    // Print to stdout
    try std.io.getStdOut().writer().print("Data: {s}\n", .{content});
}
```

## Performance Optimization

### Memory Pre-allocation
```typescript
// Pre-allocate memory for better performance
class MemoryPool {
    private buffers: ArrayBuffer[] = [];
    
    allocate(size: number): ArrayBuffer {
        if (this.buffers.length > 0) {
            return this.buffers.pop()!;
        }
        return new ArrayBuffer(size);
    }
    
    free(buffer: ArrayBuffer) {
        this.buffers.push(buffer);
    }
}
```

### Caching File Descriptors
```typescript
// Cache frequently accessed files
class FDCache {
    private cache = new Map<string, CachedFD>();
    
    get(path: string): FileDescriptor | undefined {
        const cached = this.cache.get(path);
        if (cached && !cached.expired) {
            return cached.fd;
        }
        return undefined;
    }
}
```

## Security Considerations

### Capability Enforcement
```typescript
// Validate filesystem access
function checkAccess(ns: Namespace, path: string, rights: bigint): boolean {
    // Ensure path is within namespace
    if (!ns.contains(path)) {
        return false;
    }
    
    // Check requested rights
    const available = ns.getRights(path);
    return (rights & available) === rights;
}
```

### Memory Safety
```typescript
// Bounds checking for all memory access
function validateMemoryAccess(ptr: number, len: number): void {
    if (ptr < 0 || ptr + len > memory.buffer.byteLength) {
        throw new Error("Memory access out of bounds");
    }
}
```

## Debugging Support

### Trace Logging
```typescript
// Optional WASI call tracing
if (DEBUG) {
    console.log(`WASI: fd_read(${fd}, ${iovs_len} iovecs)`);
}
```

### Error Mapping
```typescript
// Map JS errors to WASI error codes
function errorToWasi(error: Error): number {
    switch (error.name) {
        case 'ENOENT': return WASI_ERRNO_NOENT;
        case 'EACCES': return WASI_ERRNO_ACCES;
        case 'ENOTDIR': return WASI_ERRNO_NOTDIR;
        default: return WASI_ERRNO_IO;
    }
}
```

## Future Enhancements

- **WASI Preview 2**: Support for component model
- **Threads**: WebAssembly threads support
- **SIMD**: Performance optimizations
- **Better Debugging**: Source-level debugging
- **Network Sockets**: TCP/UDP support through WASI