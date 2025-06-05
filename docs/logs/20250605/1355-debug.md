# Debug Analysis: Node Service Not Mounting in /web/

**Date**: 2025-06-05 13:55  
**Issue**: The node service is not appearing in /web/ despite being added to the MapFS

## Problem Summary

After implementing Node.js support in Wanix, the `/web/node/` service is not being mounted, even though:
1. The code appears correct in `web/web.go`
2. The nodejs task type is registered and working
3. The WASM has been rebuilt multiple times

## Current State

### What's Working
- `nodejs` task type appears in `/task/new/`
- We can create nodejs tasks (got task ID 3)
- The task structure exists at `/task/3/`
- Other web services (dom, opfs, sw, vm, worker) are mounted correctly

### What's Not Working
- `/web/node/` does not exist
- The bootstrap.js file cannot be accessed
- Node.js tasks fail because they can't load bootstrap.js

## Root Cause Analysis

### Hypothesis 1: Import/Initialization Issue
The `node.New()` function might be failing silently or returning nil.

**Evidence Against**: 
- Go would panic if New() returned nil when trying to use it as fs.FS
- No panic is occurring, so it's likely not nil

### Hypothesis 2: Build Tag Issues
The service might not be included in the WASM build due to build constraints.

**Evidence For**:
- We had to add `//go:build js && wasm` to fix compilation
- Other services have these tags

**Evidence Against**:
- The package builds successfully with `GOOS=js GOARCH=wasm go list ./web/node`

### Hypothesis 3: Embed Directive Issues in WASM
The `//go:embed` directive might not work properly in WASM builds.

**Evidence For**:
- This is a known limitation in some WASM environments
- The bootstrap.js file might not be getting embedded

**Evidence Against**:
- Other services likely use embed (need to verify)

### Hypothesis 4: Interface Mismatch
The Service struct might not implement fs.FS correctly.

**Evidence For**:
- We're returning `*Service` which only has an `Open` method
- MapFS expects fs.FS interface

**Evidence Against**:
- fs.FS only requires Open(name string) (fs.File, error)
- Our Service implements this

### Hypothesis 5: Silent Error During Service Creation
The service might be failing during initialization but the error is swallowed.

**Evidence For**:
- No error handling in web.New() for service creation
- Silent failures are common in initialization code

## Deep Dive: Comparing with Working Services

### Worker Service (WORKS)
```go
workerfs := worker.New(k.Root)  // Returns special type
webfs["worker"] = workerfs       // Different pattern
```

### DOM Service (WORKS)
```go
webfs["dom"] = dom.New(k)       // Passes kernel
```

### Our Node Service (DOESN'T WORK)
```go
webfs["node"] = node.New()      // No parameters
```

## Critical Realization

Looking at the pattern, I notice:
1. Worker service gets special treatment (`workerfs` variable)
2. DOM service receives the kernel parameter
3. VM service likely has internal state
4. Our node service is too simple - just returns empty struct

## Most Likely Root Cause

**The Service struct doesn't properly implement fs.FS because it's missing the required methods or the embed isn't working in WASM context.**

## Debugging Steps to Confirm

1. **Check if embed works in WASM**
   - Create a test to see if bootstrapFS actually contains the file
   - Log the contents of bootstrapFS in the New() function

2. **Verify fs.FS implementation**
   - Ensure Open() is being called
   - Check what happens when opening "."

3. **Compare with simpler services**
   - Look at how sw or vm services work
   - They might have simpler patterns we can follow

## Potential Solutions

### Solution 1: Use MapFS Directly
Instead of a Service struct, return a MapFS directly:

```go
func New() fs.FS {
    data, _ := bootstrapFS.ReadFile("bootstrap.js")
    return fskit.MapFS{
        "bootstrap.js": fskit.Entry("bootstrap.js", 0444, data),
        "ctl": fskit.Entry("ctl", 0644, []byte{}),
    }
}
```

### Solution 2: Implement StatFS
The Service might need to implement StatFS as well:

```go
func (s *Service) Stat(name string) (fs.FileInfo, error) {
    // Implementation
}
```

### Solution 3: Don't Use Embed
Read bootstrap.js differently, perhaps as a string constant:

```go
const bootstrapJS = `(function(global) { ... })(this);`
```

### Solution 4: Debug Initialization
Add logging to see if New() is even being called:

```go
func New() *Service {
    js.Global().Get("console").Call("log", "NODE SERVICE INIT")
    return &Service{}
}
```

## Recommended Next Steps

1. **Simplify the implementation** - Use Solution 1 (MapFS directly)
2. **Add debug logging** - Confirm New() is called
3. **Test embed in WASM** - Verify embed works at all
4. **Compare with working services** - Find the minimal working pattern

## Conclusion

The most likely issue is that our Service implementation is too minimal and/or the embed directive isn't working in WASM. The quickest fix is to change the implementation to return a simple MapFS with the bootstrap.js content as a string constant rather than using embed.