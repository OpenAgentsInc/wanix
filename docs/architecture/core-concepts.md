# Core Concepts

Understanding these fundamental concepts is essential for working with Wanix.

## Everything is a File

In Wanix, following Plan 9's philosophy, all system resources are exposed as files:

- **Devices**: Mouse, keyboard, display are files you read/write
- **Services**: DOM manipulation, web workers accessed through files
- **APIs**: Network sockets, storage systems as file interfaces
- **Control**: System operations performed by writing to control files

This uniformity means you only need to learn one API - the filesystem API - to interact with the entire system.

### Example: Creating a DOM element
```bash
# Allocate a new iframe
id=$(cat /web/dom/new/iframe)

# Configure it by writing to its files
echo "src=/index.html" >> /web/dom/$id/attrs
echo "width=800" >> /web/dom/$id/attrs

# Add it to the page
echo "append-child $id" > /web/dom/body/ctl
```

## Namespaces

A namespace in Wanix is a process's view of the filesystem. Unlike traditional Unix where all processes share one filesystem view, each Wanix process has its own customizable namespace.

### Key Properties:
- **Inherited**: Child processes inherit parent's namespace by default
- **Mutable**: Can be modified with bind/unbind operations
- **Isolated**: Processes can only access what's in their namespace
- **Composable**: Build complex environments from simple pieces

### Namespace Operations

**Binding**: Make a file/directory available at a new location
```bash
# Make /web/opfs/data.txt also available as /data.txt
bind /web/opfs/data.txt /data.txt
```

**Union Binding**: Merge multiple directories
```bash
# Merge two directories into one view
bind /web/opfs/bin /bin
bind /usr/local/bin /bin  # Now /bin contains both
```

**Unbinding**: Remove something from namespace
```bash
unbind /cap  # Remove capability service access
```

## Capabilities

In Wanix, capabilities are resources that provide specific functionality. They follow the "capability security" model where possession of a capability grants access to a resource.

### The Clone Dance

Most capabilities are allocated using the Plan 9 "clone dance" pattern:

1. **Read the clone file** to allocate a new instance
2. **Get an ID** that represents your capability
3. **Access the capability** through its ID

```bash
# Allocate a web worker capability
id=$(cat /cap/new/worker)

# Use the capability
echo "start /web/opfs/worker.js" > /cap/$id/ctl

# Check its status
cat /cap/$id/status
```

### Common Capabilities

- **pickerfs**: Browser file picker integration
- **worker**: Web Worker management
- **vm**: Virtual machine instances
- **dom**: DOM element creation
- **xterm**: Terminal emulator

## File Services

File services are the building blocks of Wanix. They implement filesystem interfaces that provide specific functionality.

### Types of File Services

1. **Storage Services**
   - OPFS (Origin Private File System)
   - Memory filesystems
   - TAR archives

2. **Device Services**
   - Terminal I/O
   - Mouse/keyboard input
   - Display output

3. **API Services**
   - DOM manipulation
   - Web Workers
   - Service Workers
   - Network sockets

4. **System Services**
   - Process management (/task)
   - Capability allocation (/cap)
   - Module loading

### File Service Patterns

**Control Files**: Special files that accept commands
```bash
echo "remove" > /web/dom/$id/ctl
echo "terminate" > /web/worker/$id/ctl
```

**Data Files**: Read/write streams of data
```bash
# Write to terminal
echo "Hello" > /web/dom/$term_id/data

# Read mouse position
cat /dev/mouse
```

**Synthetic Files**: Generated content
```bash
# Process information
cat /task/$pid/status
cat /task/$pid/namespace
```

## Process Model

Wanix uses "tasks" as its process abstraction:

- **Isolated**: Each task has its own namespace
- **Lightweight**: Minimal overhead for task creation
- **Multi-runtime**: Supports WASI and x86 executables
- **Controllable**: Fine-grained control over task environment

### Task Creation Example
```bash
# Create a new task
task_id=$(cat /task/new)

# Configure its namespace
echo "unbind /cap" > /task/$task_id/ns/ctl
echo "bind /custom/root /" > /task/$task_id/ns/ctl

# Set command and start
echo "cmd /bin/myapp" > /task/$task_id/ctl
echo "start" > /task/$task_id/ctl
```

## Security Model

Wanix security is based on capability principles:

1. **No Ambient Authority**: Processes only have access to what's explicitly in their namespace
2. **Capability-Based**: Access requires possession of a capability (file service)
3. **Fine-Grained**: Control access at the file/directory level
4. **Composable**: Build exactly the environment each process needs

### Example: Sandboxed Process
```bash
# Create a task with minimal capabilities
task_id=$(cat /task/new)

# Give it only OPFS access, no DOM or network
echo "bind /web/opfs /storage" > /task/$task_id/ns/ctl
echo "bind #null /dev/null" > /task/$task_id/ns/ctl

# Start sandboxed program
echo "cmd /storage/untrusted.wasm" > /task/$task_id/ctl
echo "start" > /task/$task_id/ctl
```

## The Power of Composition

These simple primitives combine to create powerful systems:

- **Containers**: Isolated namespaces = process containers
- **Remote Resources**: Network file services = distributed systems
- **Virtual Environments**: Custom namespaces = development environments
- **Security Policies**: Capability distribution = access control

The beauty of Wanix is that complex systems emerge from these simple, composable pieces, all unified under the filesystem abstraction.