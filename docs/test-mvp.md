# Testing Node.js MVP in Wanix

This document provides step-by-step instructions for testing the Node.js MVP implementation in Wanix.

## Prerequisites

- Docker 20.10+ (for building dependencies)
- Go 1.23+ (for native builds)
- TinyGo 0.35+ (optional, for smaller WASM builds)
- A modern web browser (Chrome/Firefox/Safari)

## Building Wanix with Node.js Support

### 1. Clone and Switch to MVP Branch

```bash
git clone https://github.com/OpenAgentsInc/wanix.git
cd wanix
git checkout mvp
```

### 2. Build Dependencies (First Time Only)

This step builds all external dependencies and takes 10-30 minutes:

```bash
make deps
```

This builds:
- v86 (x86 emulator)
- Linux kernel
- Shell environment (with our new `/bin/node` script)
- WASI shim

### 3. Build Wanix

```bash
make build
```

Or for faster builds with larger WASM (better for debugging):
```bash
make wasm-go wanix
```

### 4. Start Wanix Server

```bash
./wanix serve
```

This starts a web server on http://localhost:8080

## Testing Node.js Support

### 1. Open Wanix in Browser

Navigate to http://localhost:8080 in your browser. You should see the Wanix shell.

### 2. Create a Test Script

In the Wanix shell, create a simple JavaScript file:

```bash
echo 'console.log("Hello from Node.js in Wanix!");' > hello.js
```

### 3. Run with Node

```bash
node hello.js
```

Expected output:
```
Hello from Node.js in Wanix!
```

### 4. Test Multiple Console Arguments

```bash
cat > test2.js << 'EOF'
console.log('Testing multiple arguments:', 1, true, {x: 42});
console.error('This goes to stderr');
EOF

node test2.js
```

Expected output:
```
Testing multiple arguments: 1 true {"x":42}
This goes to stderr
```

### 5. Test Process Exit

```bash
cat > test3.js << 'EOF'
console.log('Starting...');
process.exit(0);
console.log('This should not print');
EOF

node test3.js
```

Expected output:
```
Starting...
```

### 6. Test Process Object

```bash
cat > test4.js << 'EOF'
console.log('CWD:', process.cwd());
console.log('Argv:', process.argv);
process.stdout.write('Direct stdout write\n');
process.stderr.write('Direct stderr write\n');
EOF

node test4.js
```

Expected output:
```
CWD: /
Argv: ["node"]
Direct stdout write
Direct stderr write
```

## Verifying the Implementation

### Check That Node.js Task Type is Registered

```bash
ls /task/new/
```

You should see `nodejs` in the list.

### Check That Bootstrap.js is Available

```bash
cat /web/node/bootstrap.js
```

This should show the bootstrap code with console and process implementations.

### Create a Node.js Task Manually

```bash
# Allocate a nodejs task
task_id=$(cat /task/new/nodejs)

# Write a simple script
echo 'console.log("Manual task test");' > /task/$task_id/cmd

# Start the task
echo "start" > /task/$task_id/ctl

# Read output
cat /task/$task_id/fd/1
```

## Troubleshooting

### "node: not found" Error

Make sure the shell was rebuilt with the new `/bin/node` script:
```bash
ls /bin/node
```

If missing, rebuild the shell:
```bash
cd shell && make build
cd .. && make build
```

### No Output from Node

Check if the nodejs task type is registered:
```bash
ls /task/new/nodejs
```

If you get an error, the WASM module may not have the Node.js support. Rebuild:
```bash
make clean
make wasm-go
make wanix
```

### JavaScript Errors

Check the browser console (F12) for JavaScript errors. The Node.js task runs in the browser's JavaScript engine, so errors will appear there.

### Task Hangs

If a task hangs, you can check its status:
```bash
# Find the task ID
ls /task/

# Check status
cat /task/<id>/status
```

## What's Working

✅ Basic console.log and console.error  
✅ Process object with stdout, stderr, exit  
✅ Running .js files with `node` command  
✅ Exit codes propagated correctly  
✅ Multiple console arguments with object stringification  

## What's NOT Working (Yet)

❌ Real module system (require only returns empty object)  
❌ File system access (no fs module)  
❌ Async operations (no promises, setTimeout, etc.)  
❌ Command line arguments to scripts  
❌ Environment variables  
❌ Any Node.js built-in modules beyond console/process  

## Next Steps

Once basic testing is verified, the next phases would add:

1. Basic fs module for file operations
2. Simple require() for local modules
3. Async support with event loop
4. More process properties (argv, env, cwd)
5. NPM package support

## Debug Mode

For debugging, you can enable verbose logging:

1. In the browser console:
```javascript
localStorage.setItem('wanix.debug', 'true');
```

2. Reload the page and run node commands to see detailed logs.

## Summary

This MVP demonstrates that JavaScript code can execute within the Wanix environment using a Node.js-like interface. While limited, it proves the concept and provides a foundation for building a full Node.js runtime in the browser through Wanix's capability-based architecture.