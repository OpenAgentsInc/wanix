# Process Model

Wanix implements an abstract process model that supports multiple execution environments while maintaining consistent interfaces and isolation guarantees.

## Overview

The Wanix process model, called "Tasks", provides:
- **Universal abstraction** for both WASI and x86 processes
- **Per-process namespaces** for isolation
- **Resource management** through file interfaces
- **Flexible IPC** through shared namespace portions

## Task Architecture

```
┌─────────────────────────────────────┐
│            Task Manager             │
│   (task creation, lifecycle)        │
└──────────┬────────────┬─────────────┘
           │            │
    ┌──────▼─────┐ ┌────▼──────┐
    │ WASI Task  │ │ x86 Task  │
    │  (Wasm)    │ │  (Linux)  │
    └──────┬─────┘ └────┬──────┘
           │            │
    ┌──────▼────────────▼──────┐
    │    Task Resources        │
    │ • Namespace              │
    │ • File descriptors       │
    │ • Environment vars       │
    │ • Working directory      │
    └──────────────────────────┘
```

## Core Components

### Task Service (`task/service.go`)

The task service manages all processes in the system:

```go
type TaskService struct {
    tasks    map[string]*Task
    nextPID  int
    mu       sync.RWMutex
}

type Task struct {
    ID        string
    PID       int
    Type      TaskType  // WASI or Linux
    Namespace *vfs.Namespace
    Env       []string
    Cwd       string
    Status    TaskStatus
    ExitCode  int
}
```

### Task Types

#### 1. WASI Tasks
- Run WebAssembly modules compiled for WASI
- Direct integration with browser WebAssembly runtime
- Lightweight and fast startup
- Full namespace access through WASI APIs

#### 2. Linux/x86 Tasks  
- Run via v86 emulator with custom Linux kernel
- Support for existing Linux binaries
- 9P filesystem bridge to Wanix namespace
- Higher overhead but broader compatibility

## Task Lifecycle

### 1. Creation

Tasks are created through the clone interface:
```bash
# Allocate new task
task_id=$(cat /task/new)

# Returns a task ID for configuration
```

The new task inherits:
- Parent's namespace (can be modified)
- Parent's environment variables
- Parent's working directory
- Standard file descriptors

### 2. Configuration

Before starting, tasks can be configured:

```bash
# Set command
echo "cmd /bin/myapp arg1 arg2" > /task/$task_id/ctl

# Modify namespace
echo "unbind /sensitive" > /task/$task_id/ns/ctl
echo "bind /custom /mnt" > /task/$task_id/ns/ctl

# Set environment
echo "PATH=/bin:/usr/bin" > /task/$task_id/env
echo "HOME=/home/user" > /task/$task_id/env

# Set working directory  
echo "/home/user" > /task/$task_id/cwd
```

### 3. Execution

Start the configured task:
```bash
echo "start" > /task/$task_id/ctl
```

Monitor execution:
```bash
# Check status
cat /task/$task_id/status  # running/stopped/exited

# Read output
cat /task/$task_id/fd/1  # stdout
cat /task/$task_id/fd/2  # stderr

# Wait for completion
cat /task/$task_id/wait  # Blocks until exit
```

### 4. Termination

Tasks can be terminated:
```bash
# Send signal
echo "kill 15" > /task/$task_id/ctl  # SIGTERM
echo "kill 9" > /task/$task_id/ctl   # SIGKILL

# Check exit code
cat /task/$task_id/exitcode
```

## File Descriptor Management

Each task has its own file descriptor table:

```
/task/$id/fd/
├── 0     # stdin
├── 1     # stdout  
├── 2     # stderr
├── 3     # custom file
└── ...
```

File descriptors can be:
- Inherited from parent
- Redirected to files
- Piped between processes
- Connected to devices

### Example: I/O Redirection
```bash
# Create task with redirected I/O
task_id=$(cat /task/new)

# Redirect stdout to file
echo "dup /tmp/output.log" > /task/$task_id/fd/1/ctl

# Pipe stdin from another process
echo "pipe $other_task/fd/1" > /task/$task_id/fd/0/ctl
```

## Namespace Management

Each task's namespace is independently configurable:

### Namespace Operations

```go
// In task creation
task.Namespace = parent.Namespace.Fork()

// Namespace modification
task.Namespace.Bind(oldpath, newpath, flags)
task.Namespace.Unbind(path)
task.Namespace.Mount(fs, path)
```

### Namespace File Interface

```
/task/$id/ns/
├── ctl      # Control file for modifications
├── mounts   # Current mount table
├── binds    # Current bind table
└── dump     # Full namespace dump
```

## Inter-Process Communication

### 1. Shared Namespace Portions
```bash
# Create shared directory
mkdir /tmp/shared

# Both tasks can access
echo "bind /tmp/shared /shared" > /task/$task1/ns/ctl
echo "bind /tmp/shared /shared" > /task/$task2/ns/ctl
```

### 2. Pipes
```bash
# Create pipe
pipe_id=$(cat /cap/new/pipe)

# Connect processes
echo "dup /cap/$pipe_id/w" > /task/$task1/fd/1/ctl
echo "dup /cap/$pipe_id/r" > /task/$task2/fd/0/ctl
```

### 3. File-based IPC
```bash
# Signal files, lock files, message queues
# All implemented as regular files
echo "ready" > /tmp/signals/proc1
while [ ! -f /tmp/signals/proc2 ]; do sleep 1; done
```

## Resource Management

### Memory Limits
```bash
# Set memory limit (WASI tasks)
echo "memlimit 100M" > /task/$task_id/ctl
```

### CPU Scheduling
```bash
# Set priority
echo "nice 10" > /task/$task_id/ctl
```

### Resource Accounting
```bash
# View resource usage
cat /task/$task_id/stat/mem
cat /task/$task_id/stat/cpu
cat /task/$task_id/stat/io
```

## Security Boundaries

### 1. Namespace Isolation
- Tasks cannot access files outside their namespace
- No ambient authority - explicit capability required
- Parent controls child's initial namespace

### 2. Resource Isolation  
- Separate file descriptor tables
- Independent environment variables
- Isolated working directories

### 3. Communication Control
- IPC requires explicit namespace sharing
- No implicit channels between tasks
- All communication through file interfaces

## Implementation Details

### WASI Integration (`external/wasi/`)

WASI tasks use a JavaScript shim that:
1. Translates WASI calls to Wanix VFS operations
2. Manages memory and module instantiation
3. Handles async I/O through Web Workers

```typescript
// Simplified WASI shim
class WASITask {
    constructor(namespace: Namespace) {
        this.namespace = namespace;
        this.memory = new WebAssembly.Memory({...});
    }
    
    fd_read(fd: number, iovs: number, iovsLen: number) {
        const file = this.namespace.getFileDescriptor(fd);
        // Read from file into WASM memory
    }
}
```

### Linux/x86 Integration (`web/vm/`)

Linux tasks run in v86 emulator with:
1. Custom minimal Linux kernel
2. 9P root filesystem connected to Wanix
3. Serial console for I/O
4. Virtio devices for better performance

```go
// VM task creation
func createVMTask(ns *vfs.Namespace) *VMTask {
    vm := &VMTask{
        v86: v86.New(),
        fs: p9kit.NewServer(ns),
    }
    
    // Mount Wanix namespace as 9P
    vm.Mount9P("/", vm.fs)
    
    return vm
}
```

## Best Practices

1. **Minimize Namespace**: Give tasks only required capabilities
2. **Resource Cleanup**: Always cleanup file descriptors
3. **Error Handling**: Check exit codes and status
4. **Isolation**: Don't share more namespace than needed
5. **Monitoring**: Use stat files to track resource usage

## Future Enhancements

- **Checkpoint/Restore**: Save and restore task state
- **Remote Tasks**: Run tasks on other machines
- **GPU Access**: WebGPU integration for compute tasks
- **Better Scheduling**: Priority-based CPU scheduling
- **Resource Quotas**: Enforce resource limits