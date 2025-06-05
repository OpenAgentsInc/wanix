# File Services Reference

Complete reference for all built-in file services in Wanix.

## System Services

### Capability Service (`/cap`)

The capability service manages system resources and access control.

#### Structure
```
/cap/
├── new/              # Capability allocation
│   ├── tarfs         # TAR filesystem
│   ├── tmpfs         # Temporary filesystem
│   ├── loopback      # Namespace loopback
│   └── ...           # Other capabilities
└── <id>/             # Capability instances
    ├── ctl           # Control file
    ├── type          # Capability type
    └── ...           # Type-specific files
```

#### Allocation
```bash
# Allocate new capability
id=$(cat /cap/new/tmpfs)

# Access capability
ls /cap/$id/
```

#### Built-in Capabilities

**tmpfs** - In-memory filesystem
```
/cap/<id>/
├── ctl          # Commands: limit, clear, snapshot
├── type         # "tmpfs"
├── size         # Current size in bytes
└── fs/          # Filesystem root
```

**tarfs** - TAR archive filesystem
```
/cap/<id>/
├── ctl          # Commands: load
├── type         # "tarfs"
├── data         # Write TAR data here
└── fs/          # Filesystem root (after load)
```

**loopback** - Namespace view
```
/cap/<id>/
├── ctl          # Commands: bind
├── type         # "loopback"
└── root/        # Loopback root
```

### Task Service (`/task`)

Process management and control.

#### Structure
```
/task/
├── new              # Task allocation
└── <id>/            # Task instances
    ├── ctl          # Control file
    ├── status       # Current status
    ├── exitcode     # Exit code (when done)
    ├── cmd          # Command line
    ├── env          # Environment variables
    ├── cwd          # Working directory
    ├── ns/          # Namespace control
    │   ├── ctl      # Namespace commands
    │   └── dump     # Namespace dump
    └── fd/          # File descriptors
        ├── 0        # stdin
        ├── 1        # stdout
        └── 2        # stderr
```

#### Task Lifecycle
```bash
# Create task
task_id=$(cat /task/new)

# Configure
echo "cmd /bin/program arg1 arg2" > /task/$task_id/ctl
echo "VAR=value" >> /task/$task_id/env

# Start
echo "start" > /task/$task_id/ctl

# Monitor
cat /task/$task_id/status

# Terminate
echo "kill 9" > /task/$task_id/ctl
```

#### Control Commands
- `cmd <command> [args...]` - Set command to run
- `start` - Start the task
- `kill <signal>` - Send signal (9, 15, etc.)
- `suspend` - Suspend execution
- `resume` - Resume execution

## Web Services

### DOM Service (`/web/dom`)

Document Object Model manipulation.

#### Structure
```
/web/dom/
├── body/            # Document body
│   ├── ctl          # Control commands
│   └── attrs        # Attributes
├── style            # Page styles (append CSS)
├── new/             # Element creation
│   ├── div
│   ├── span
│   ├── iframe
│   ├── img
│   ├── xterm
│   └── ...
└── <id>/            # Element instances
    ├── ctl          # Control file
    ├── attrs        # Element attributes
    ├── style        # Inline styles
    ├── class        # CSS classes
    ├── inner        # innerHTML
    └── data         # Special data (xterm)
```

#### Element Creation
```bash
# Create element
elem_id=$(cat /web/dom/new/div)

# Set attributes
echo "id=myDiv" >> /web/dom/$elem_id/attrs
echo "class=container" >> /web/dom/$elem_id/attrs

# Add to page
echo "append-child $elem_id" > /web/dom/body/ctl
```

#### Control Commands
- `append-child <id>` - Add child element
- `remove-child <id>` - Remove child element
- `insert-before <id> <ref>` - Insert before reference
- `remove` - Remove element from DOM
- `focus` - Focus element
- `click` - Trigger click event

#### Special Elements

**xterm** - Terminal emulator
```
/web/dom/<id>/
├── ctl          # Commands: reset, clear
├── data         # Terminal I/O stream
├── cols         # Column count
└── rows         # Row count
```

### Worker Service (`/web/worker`)

Web Worker management.

#### Structure
```
/web/worker/
├── new              # Worker allocation
└── <id>/            # Worker instances
    ├── ctl          # Control file
    ├── status       # Current status
    ├── url          # Script URL
    ├── send         # Send message
    └── recv         # Receive messages
```

#### Worker Control
```bash
# Create worker
worker_id=$(cat /web/worker/new)

# Start with script
echo "start /scripts/worker.js" > /web/worker/$worker_id/ctl

# Send message
echo '{"type":"task","data":[1,2,3]}' > /web/worker/$worker_id/send

# Read messages
cat /web/worker/$worker_id/recv
```

#### Control Commands
- `start <url>` - Start worker with script
- `terminate` - Terminate worker
- `suspend` - Suspend execution
- `resume` - Resume execution

### Service Worker (`/web/sw`)

Service Worker configuration.

#### Structure
```
/web/sw/
├── ctl              # Control file
├── status           # Registration status
├── scope            # Service worker scope
└── routes           # URL routing rules
```

#### Control Commands
- `register <url>` - Register service worker
- `unregister` - Unregister service worker
- `update` - Force update check
- `skipWaiting` - Activate immediately

#### Route Configuration
```bash
# Route namespace paths to URLs
echo "route /app/* /web/opfs/app" >> /web/sw/routes

# Enable caching
echo "cache /static/*" >> /web/sw/routes
```

### OPFS Service (`/web/opfs`)

Origin Private File System - browser storage.

#### Structure
```
/web/opfs/           # Root of OPFS
├── file.txt         # Regular files
├── dir/             # Directories
└── ...              # User content
```

#### Features
- Persistent storage
- Full filesystem semantics
- Survives page reload
- Private to origin

### WebSocket Service (`/web/ws`)

WebSocket connections.

#### Structure
```
/web/ws/
├── new              # Connection allocation
└── <id>/            # Connection instances
    ├── ctl          # Control file
    ├── status       # Connection status
    ├── url          # WebSocket URL
    ├── send         # Send data
    └── recv         # Receive data
```

#### WebSocket Control
```bash
# Create connection
ws_id=$(cat /web/ws/new)

# Connect
echo "connect wss://example.com/ws" > /web/ws/$ws_id/ctl

# Send message
echo "Hello, server!" > /web/ws/$ws_id/send

# Receive messages
cat /web/ws/$ws_id/recv
```

#### Control Commands
- `connect <url>` - Connect to WebSocket server
- `close [code] [reason]` - Close connection
- `ping` - Send ping frame

### File System Access (`/web/fsa`)

Native file picker integration (Chrome only currently).

#### Usage
```bash
# Allocate picker capability
picker_id=$(cat /cap/new/pickerfs)

# Show directory picker
echo "mount" > /cap/$picker_id/ctl

# Access picked directory
ls /cap/$picker_id/mount/
```

## Storage Services

### Memory FS
Built-in memory filesystem available through tmpfs capability.

### TAR FS
Read-only filesystem from TAR archives through tarfs capability.

## Special Files

### Null Device (`#null`)
Discards all writes, returns EOF on read.
```bash
echo "discarded" > "#null"
```

### Shell Root (`#shell`)
Built-in shell environment with BusyBox.
```bash
ls "#shell/bin"
```

### Current Namespace (`#dot`)
Self-reference to current namespace.
```bash
bind "#dot" /mnt/ns
```

## File Patterns

### Control Files
Most services use control files for operations:
- Write commands as text lines
- One command per write
- Commands are space-separated
- Arguments may be quoted

### Status Files
Read-only files providing current state:
- Human-readable format
- Updated automatically
- Poll for changes

### Data Streams
Bidirectional communication files:
- Read for incoming data
- Write for outgoing data
- May block on read

### Clone Files
Allocate new resources by reading:
- Returns unique identifier
- Creates new instance
- Instance appears as directory

## Error Handling

### Common Errors
- `ENOENT` - File/service not found
- `EACCES` - Permission denied
- `EINVAL` - Invalid argument
- `EBUSY` - Resource busy
- `EEXIST` - Already exists

### Error Responses
Control files return errors as text:
```
error: invalid command
error: missing argument
error: operation failed: <reason>
```

## Best Practices

1. **Check service availability** before use
2. **Close resources** when done
3. **Handle errors** appropriately
4. **Use unique IDs** for allocated resources
5. **Follow conventions** for control commands
6. **Read documentation** in service directories

## Service Discovery

List available services:
```bash
# System services
ls /

# Web services  
ls /web

# Capabilities
ls /cap/new

# Check service type
cat /cap/$id/type
```