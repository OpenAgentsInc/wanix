# Comprehensive Analysis: Node.js MVP Implementation in Wanix
## Date: 2025-06-05 21:15

## Overview

This document provides a comprehensive analysis of the ongoing effort to implement Node.js MVP support in Wanix. The work has involved implementing filesystem Create methods across the entire VFS stack and debugging complex namespace resolution issues.

## Context and Goal

We are implementing Node.js support in Wanix as part of the MVP branch. The goal is to enable running JavaScript files using a `node` command within the Wanix shell environment. The implementation follows Wanix's "everything is a file" philosophy, where Node.js tasks are exposed through the filesystem at `/task/new/nodejs`.

### Key Requirements (from docs/test-mvp.md)
1. Run JavaScript files with `node hello.js` command in the Wanix shell
2. Support basic console.log/console.error output
3. Support process.exit() functionality
4. Provide process.stdout/stderr write capabilities
5. All execution happens in the browser's JavaScript engine, not a separate runtime

## Architecture Overview

### 1. Task System
- Wanix uses a task system where processes are represented as "Resources"
- Each task has a namespace and exposes control files: `cmd`, `ctl`, `dir`, `exit`, `fd/*`
- Tasks are created via `/task/new/<type>` and appear as `/task/<id>/`
- The task service manages all tasks and their lifecycles

### 2. VFS (Virtual File System)
- Everything in Wanix is exposed as files through VFS
- Namespaces (NS) manage bindings between paths and filesystems
- Multiple filesystem types: MapFS, UnionFS, MemFS, OpenFunc, etc.
- Filesystems can be composed and bound to different paths

### 3. Node.js Integration
- `NodeTask` type registered with the task service
- `/bin/node` shell script that creates nodejs tasks
- `/web/node/bootstrap.js` provides Node.js globals (console, process)
- JavaScript execution happens in browser via script evaluation

## Implementation Details

### 1. NodeTask Implementation (task/node_task.go)
```go
type NodeTask struct {
    resource *Resource
    script   string
}
```
- Reads JavaScript from `cmd` file
- Loads bootstrap.js for Node.js environment
- Executes script in browser context
- Pipes output to task's file descriptors

### 2. Shell Script (/bin/node)
```bash
#!/bin/sh
# Create nodejs task
task_id=$(cat /task/new/nodejs)
# Write script content
cat "$1" > /task/$task_id/cmd
# Start execution
echo "start" > /task/$task_id/ctl
# Wait and show output
cat /task/$task_id/fd/1 &
cat /task/$task_id/fd/2 >&2
wait
exit $(cat /task/$task_id/exit 2>/dev/null || echo 0)
```

### 3. Bootstrap Environment (web/node/bootstrap.js)
Provides Node.js-compatible globals:
- `console.log()`, `console.error()`, `console.warn()`
- `process.stdout.write()`, `process.stderr.write()`
- `process.exit()`
- `process.cwd()`, `process.argv`
- Basic `require()` stub

## The Create Method Problem

### Root Cause Discovery
The shell uses output redirection (`>`) which internally uses `open(..., O_CREAT)`. This requires filesystems to implement the `fs.CreateFS` interface. Without Create support, the shell couldn't write to task control files.

### Implementation Cascade
We had to implement Create methods across the entire filesystem stack:

1. **task.Resource** (task/proc.go)
   - Handles synthetic files like cmd, ctl, dir, exit
   - Delegates to OpenContext for these writable files

2. **fskit.MapFS** (fs/fskit/mapfs.go)
   - Maps string paths to filesystems
   - Needed to delegate Create to underlying filesystems

3. **vfs.NS** (vfs/vfs.go)
   - Namespace that manages path bindings
   - Had to handle Create for bound paths
   - Fixed stack overflow by avoiding recursive resolution

4. **web/dom.Element** (web/dom/element.go)
   - DOM elements exposed as files
   - Delegates Create to Open

5. **fskit.UnionFS** (fs/fskit/unionfs.go)
   - Combines multiple filesystems
   - Tries each filesystem in order for Create

6. **fskit.OpenFunc** (fs/fskit/openfunc.go)
   - Function-based filesystem
   - Delegates Create to the open function

7. **task.Service** (task/service.go)
   - The task management service itself
   - Routes Create calls to appropriate task resources

## Current Problem: Namespace Resolution

### The Issue
When `bootShell()` runs to set up the Wanix shell environment, it tries to bind paths for terminal I/O:
```javascript
await w.writeFile("task/1/ctl", "bind web/dom/1/data web/vm/1/ttyS0");
```

This fails with: `proc.go:129: open 1/data: file does not exist`

### Analysis
The debug logs show:
```
Resource.Bind: srcPath="web/dom/1/data", dstPath="web/vm/1/ttyS0"
Resource.Bind: resolved fsys=fskit.MapFS, resolvedPath="1/data"
```

The path "web/dom/1/data" is being resolved to just "1/data" - the "web/dom/" prefix is being stripped incorrectly.

### Root Cause
The issue is in how `fs.Resolve` works with the namespace bindings. When resolving "web/dom/1/data":
1. It finds "web" is bound to a MapFS
2. It strips "web/" and continues with "dom/1/data"
3. Within that MapFS, it finds "dom" and strips it
4. The final resolved path becomes just "1/data"

But "1/data" doesn't exist in the MapFS - it should be looking for "dom/1/data" or the full path should be preserved differently.

## Architecture Challenges

### 1. Complex Resolution Chain
The path resolution goes through multiple layers:
- NS (namespace) → MapFS → potentially another filesystem
- Each layer can transform the path
- Difficult to debug when paths are incorrectly transformed

### 2. Create vs Open Semantics
- Shell expects Create to work like open(..., O_CREAT)
- Some filesystems have synthetic files that always exist
- Need to handle both "create new file" and "open existing for write"

### 3. Context Propagation
- Task context needs to flow through filesystem operations
- Some operations need to know which task is making the request
- Context is lost or not properly propagated in some cases

### 4. Circular Dependencies
- Task service creates namespaces
- Namespaces need to bind the task service
- Can create infinite loops if not careful (e.g., ResolveFS returning self)

## What We've Changed

### 1. Added Create Methods
- Implemented fs.CreateFS interface across 7+ filesystem types
- Each implementation handles Create appropriately for its type
- Added interface assertions to ensure compliance

### 2. Fixed Stack Overflow
- Task service's ResolveFS was causing infinite recursion
- Fixed by returning service itself for "." path
- Added special handling in Open/Stat for root directory

### 3. Improved Error Messages
- Enhanced error messages to show exactly what failed
- Added debug logging throughout the stack
- Server version string updates to track changes

### 4. Fixed Resource.Bind
- Was incorrectly passing task's namespace as source
- Now properly resolves source paths before binding
- Still having issues with path resolution

## Current Status

### Working
✅ Create methods implemented across entire VFS stack
✅ Task service properly returns itself for root resolution
✅ Node.js task type is registered and available
✅ Bootstrap.js is accessible at /web/node/
✅ Basic task creation and management

### Not Working
❌ Shell initialization fails due to bind path resolution
❌ "web/dom/1/data" incorrectly resolves to "1/data"
❌ Can't establish terminal I/O bindings
❌ Therefore can't get to shell prompt to test Node.js

## Next Steps

1. **Fix Path Resolution**: The immediate blocker is fixing how paths are resolved through the namespace and MapFS layers. The "web/dom/" prefix shouldn't be stripped when resolving "web/dom/1/data".

2. **Test Node.js**: Once the shell loads properly, test the Node.js implementation with the examples from docs/test-mvp.md.

3. **Debug Remaining Issues**: There will likely be additional issues once we can actually run Node.js commands.

## Conceptual Understanding

### Wanix's Unique Approach
- Everything is a file, including processes, DOM elements, and web APIs
- Capabilities control access to resources
- Namespaces provide isolation between tasks
- The browser is the OS - JavaScript execution happens natively

### The Node.js Integration
- Not a full Node.js runtime, but a compatibility layer
- Leverages browser's JavaScript engine directly
- File descriptors map to browser constructs (console, DOM, etc.)
- Provides familiar Node.js interface within Wanix's model

### Why This Is Hard
1. **Impedance Mismatch**: Unix-like filesystem semantics in a browser environment
2. **Layered Abstractions**: Multiple filesystem types composed together
3. **Path Resolution**: Complex binding and resolution rules
4. **Debugging**: Errors can originate from many layers deep

## Summary

The Node.js MVP implementation is architecturally sound but blocked by a path resolution bug in the namespace binding system. The core infrastructure is in place - tasks can be created, JavaScript can be executed, and I/O is properly wired. The immediate issue is that the shell initialization fails because it can't bind the terminal I/O paths due to incorrect path resolution in the VFS layer. Once this is fixed, we should be able to test the actual Node.js functionality.