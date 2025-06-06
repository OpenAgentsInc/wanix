# VM ttyS0 Binding Fix
## Date: 2025-06-05 23:00

### Issue
After fixing VFS resolution issues, the shell prompt never loads. The error shows:
```
vm start: no ttyS0 file <nil>
```

The VM was looking for ttyS0 in its own internal filesystem rather than accessing the bound resource from the namespace.

### Root Cause
The VM implementation in `web/vm/vm.go` had several issues:
1. Syntax errors in the `makeCtlCommand` method
2. Referenced undefined `ctx` variable
3. Orphaned code outside the command definition
4. Incorrect approach to handling ttyS0 binding

### Fix Applied
1. Fixed syntax errors in `makeCtlCommand` method
2. Changed approach - VM now looks for ttyS0 in its filesystem context using `fs.Origin`
3. Removed the ttyS0 field from VM struct since bindings are handled at namespace level
4. Updated ResolveFS to be simpler and let namespace handle binding resolution

### Key Changes

**web/vm/vm.go**:
- Fixed makeCtlCommand syntax and ctx variable reference
- Changed to use `fs.Origin(ctx.Context)` to find ttyS0 in current filesystem context
- Added proper logging for debugging ttyS0 connection
- Simplified ResolveFS implementation

### Result
The VM now properly looks for ttyS0 in the namespace context where it's bound, rather than trying to store it internally. This should allow the shell to initialize correctly when bootShell binds the xterm DOM element to the VM's serial port.

### Build Status
Successfully rebuilt with:
```bash
make wasm-go wanix
```
Updated server version to v23:00.