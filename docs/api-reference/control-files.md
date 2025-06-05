# Control Files Reference

Control files (`ctl`) are the primary mechanism for interacting with Wanix services. This reference documents the standard patterns and all available commands.

## Overview

Control files accept text commands that perform operations on services. Commands follow a consistent format across all services.

## Command Format

### Basic Syntax
```
COMMAND [ARG1] [ARG2] ... [ARGN]
```

- Commands are single words (lowercase)
- Arguments are space-separated
- One command per write operation
- Commands are processed synchronously unless noted

### Quoting Rules
```bash
# Simple arguments
echo "set key value" > /service/ctl

# Arguments with spaces
echo 'set key "value with spaces"' > /service/ctl

# Single quotes preserve everything
echo 'set key '\''value with quote'\''' > /service/ctl
```

### Response Format
- **Success**: Empty response or command-specific output
- **Error**: Text starting with "error: " followed by description
- **Async**: May return immediately, check status separately

## System Service Commands

### Capability Service (`/cap/<id>/ctl`)

#### tmpfs Commands
| Command | Arguments | Description |
|---------|-----------|-------------|
| `limit` | `<size>` | Set size limit (e.g., "10M", "1G") |
| `clear` | - | Remove all files |
| `snapshot` | `<path>` | Save snapshot to path |
| `restore` | `<path>` | Restore from snapshot |

Example:
```bash
echo "limit 100M" > /cap/$tmpfs_id/ctl
echo "clear" > /cap/$tmpfs_id/ctl
```

#### tarfs Commands
| Command | Arguments | Description |
|---------|-----------|-------------|
| `load` | - | Load TAR data from data file |
| `verify` | - | Verify TAR integrity |

Example:
```bash
cat archive.tar > /cap/$tarfs_id/data
echo "load" > /cap/$tarfs_id/ctl
```

#### loopback Commands
| Command | Arguments | Description |
|---------|-----------|-------------|
| `bind` | `<path>` | Set root path for loopback |
| `filter` | `<pattern>` | Set path filter pattern |

### Task Service (`/task/<id>/ctl`)

#### Task Control
| Command | Arguments | Description |
|---------|-----------|-------------|
| `cmd` | `<cmd> [args...]` | Set command to execute |
| `start` | - | Start task execution |
| `kill` | `<signal>` | Send signal (9, 15, etc.) |
| `suspend` | - | Suspend task execution |
| `resume` | - | Resume suspended task |
| `wait` | - | Wait for task completion |

Example:
```bash
echo "cmd /bin/ls -la" > /task/$id/ctl
echo "start" > /task/$id/ctl
echo "kill 15" > /task/$id/ctl  # SIGTERM
```

#### Namespace Control (`/task/<id>/ns/ctl`)
| Command | Arguments | Description |
|---------|-----------|-------------|
| `bind` | `<old> <new> [flags]` | Bind path in namespace |
| `unbind` | `<path>` | Remove binding |
| `mount` | `<service> <path>` | Mount service |
| `unmount` | `<path>` | Unmount path |
| `clear` | - | Clear all bindings |

Bind flags:
- `-r` - Replace existing
- `-a` - Append (union after)
- `-b` - Before (union before)

Example:
```bash
echo "bind /web/opfs /data" > /task/$id/ns/ctl
echo "unbind /cap" > /task/$id/ns/ctl
```

## Web Service Commands

### DOM Service (`/web/dom/<id>/ctl`)

#### Element Commands
| Command | Arguments | Description |
|---------|-----------|-------------|
| `append-child` | `<child-id>` | Add child element |
| `remove-child` | `<child-id>` | Remove child element |
| `insert-before` | `<child-id> <ref-id>` | Insert before reference |
| `replace-child` | `<new-id> <old-id>` | Replace child element |
| `remove` | - | Remove from DOM |
| `focus` | - | Focus element |
| `blur` | - | Blur element |
| `click` | - | Trigger click event |
| `scroll` | `<x> <y>` | Scroll to position |

Example:
```bash
echo "append-child $child_id" > /web/dom/body/ctl
echo "remove" > /web/dom/$elem_id/ctl
echo "focus" > /web/dom/$input_id/ctl
```

#### Special Elements

**body** commands:
| Command | Arguments | Description |
|---------|-----------|-------------|
| `append-child` | `<id>` | Add to body |
| `remove-child` | `<id>` | Remove from body |
| `clear` | - | Remove all children |

**xterm** commands:
| Command | Arguments | Description |
|---------|-----------|-------------|
| `reset` | - | Reset terminal |
| `clear` | - | Clear screen |
| `resize` | `<cols> <rows>` | Resize terminal |

### Worker Service (`/web/worker/<id>/ctl`)

| Command | Arguments | Description |
|---------|-----------|-------------|
| `start` | `<script-url>` | Start worker with script |
| `terminate` | - | Terminate worker |
| `postMessage` | `<json>` | Send message to worker |
| `suspend` | - | Suspend execution |
| `resume` | - | Resume execution |

Example:
```bash
echo "start /scripts/worker.js" > /web/worker/$id/ctl
echo 'postMessage {"cmd":"process","data":[1,2,3]}' > /web/worker/$id/ctl
echo "terminate" > /web/worker/$id/ctl
```

### Service Worker (`/web/sw/ctl`)

| Command | Arguments | Description |
|---------|-----------|-------------|
| `register` | `<script-url> [scope]` | Register service worker |
| `unregister` | - | Unregister current worker |
| `update` | - | Check for updates |
| `skipWaiting` | - | Skip waiting phase |
| `claim` | - | Claim all clients |
| `route` | `<pattern> <target>` | Add URL route |
| `unroute` | `<pattern>` | Remove URL route |
| `cache` | `<pattern>` | Enable caching for pattern |
| `uncache` | `<pattern>` | Disable caching |

Example:
```bash
echo "register /sw.js /" > /web/sw/ctl
echo "route /app/* /web/opfs/app" > /web/sw/ctl
echo "cache /static/*" > /web/sw/ctl
```

### WebSocket Service (`/web/ws/<id>/ctl`)

| Command | Arguments | Description |
|---------|-----------|-------------|
| `connect` | `<url>` | Connect to WebSocket server |
| `close` | `[code] [reason]` | Close connection |
| `send` | `<data>` | Send text message |
| `sendBinary` | - | Send binary (from data file) |
| `ping` | `[data]` | Send ping frame |

Example:
```bash
echo "connect wss://example.com/socket" > /web/ws/$id/ctl
echo "send Hello, Server!" > /web/ws/$id/ctl
echo "close 1000 Goodbye" > /web/ws/$id/ctl
```

## Common Patterns

### Resource Allocation
```bash
# 1. Allocate resource
resource_id=$(cat /service/new)

# 2. Configure resource
echo "configure ..." > /service/$resource_id/ctl

# 3. Use resource
# ...

# 4. Clean up (if needed)
echo "cleanup" > /service/$resource_id/ctl
```

### Status Checking
```bash
# Check status before operation
status=$(cat /service/$id/status)
if [ "$status" = "ready" ]; then
    echo "start" > /service/$id/ctl
fi

# Poll for completion
while [ "$(cat /task/$id/status)" = "running" ]; do
    sleep 1
done
```

### Error Handling
```bash
# Capture command output
result=$(echo "command arg" > /service/ctl 2>&1)
if echo "$result" | grep -q "^error:"; then
    echo "Command failed: $result"
    exit 1
fi
```

### Batch Operations
```bash
# Multiple commands to same service
{
    echo "cmd1 arg1"
    echo "cmd2 arg2"
    echo "cmd3 arg3"
} > /service/ctl

# Or use a here document
cat > /service/ctl << EOF
cmd1 arg1
cmd2 arg2
cmd3 arg3
EOF
```

## Best Practices

### 1. Command Validation
Always validate arguments before sending:
```bash
if [ -z "$arg" ]; then
    echo "Error: argument required" >&2
    exit 1
fi
echo "command $arg" > /service/ctl
```

### 2. Idempotency
Design commands to be idempotent when possible:
```bash
# Good: Can be run multiple times
echo "set-state active" > /service/ctl

# Bad: Multiple runs cause issues
echo "increment-counter" > /service/ctl
```

### 3. Atomic Operations
Use single writes for atomic operations:
```bash
# Good: Atomic operation
echo "move $src $dst" > /service/ctl

# Bad: Non-atomic
echo "copy $src $tmp" > /service/ctl
echo "delete $src" > /service/ctl
echo "move $tmp $dst" > /service/ctl
```

### 4. Resource Cleanup
Always clean up allocated resources:
```bash
# Allocate
id=$(cat /service/new)

# Use with cleanup
trap 'echo "cleanup" > /service/$id/ctl' EXIT

# ... do work ...
# Cleanup happens automatically
```

### 5. Help Discovery
Many control files support help:
```bash
# General help
echo "help" > /service/ctl

# Command-specific help
echo "help command" > /service/ctl
```

## Debugging

### Command Tracing
Enable debug mode to see command processing:
```bash
export WANIX_DEBUG=1
echo "command" > /service/ctl
# Check logs for processing details
```

### Dry Run
Some services support dry-run mode:
```bash
echo "command --dry-run" > /service/ctl
```

### Command History
Keep a log of commands for debugging:
```bash
echo "command" | tee -a commands.log > /service/ctl
```

## Future Extensions

Planned control file enhancements:
- Async command tokens for tracking
- Command pipelining
- Transaction support
- Bulk operations
- Command versioning