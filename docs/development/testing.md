# Testing

This guide covers testing strategies and best practices for Wanix development.

## Overview

Wanix uses a multi-layered testing approach:
- **Unit tests** for individual components
- **Integration tests** for subsystem interactions
- **Browser tests** for web-specific features
- **Performance tests** for optimization

## Running Tests

### Basic Test Commands

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with detailed coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./fs/fskit

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestMemFS ./fs/fskit
```

### Test Organization

Tests follow Go conventions:
```
package/
├── file.go          # Implementation
├── file_test.go     # Tests in same package
└── file_ext_test.go # Tests in external package
```

## Unit Testing

### Testing File Services

```go
package myservice_test

import (
    "io"
    "testing"
    "github.com/tractordev/wanix/myservice"
)

func TestServiceOpen(t *testing.T) {
    svc := myservice.New()
    
    tests := []struct {
        name    string
        path    string
        want    string
        wantErr bool
    }{
        {"root", "", "", true},
        {"hello", "hello", "Hello, World!", false},
        {"missing", "notfound", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            f, err := svc.Open(tt.path)
            if (err != nil) != tt.wantErr {
                t.Errorf("Open() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if err == nil {
                defer f.Close()
                data, _ := io.ReadAll(f)
                if string(data) != tt.want {
                    t.Errorf("got %q, want %q", data, tt.want)
                }
            }
        })
    }
}
```

### Testing Control Files

```go
func TestControlCommands(t *testing.T) {
    svc := myservice.New()
    
    // Open control file
    ctl, err := svc.Open("ctl")
    if err != nil {
        t.Fatal(err)
    }
    defer ctl.Close()
    
    // Test set command
    _, err = ctl.Write([]byte("set key1 value1\n"))
    if err != nil {
        t.Errorf("set command failed: %v", err)
    }
    
    // Test get command
    _, err = ctl.Write([]byte("get key1\n"))
    if err != nil {
        t.Errorf("get command failed: %v", err)
    }
    
    // Read response
    resp := make([]byte, 100)
    n, _ := ctl.Read(resp)
    if string(resp[:n]) != "value1\n" {
        t.Errorf("unexpected response: %q", resp[:n])
    }
}
```

### Testing Concurrent Access

```go
func TestConcurrentAccess(t *testing.T) {
    svc := myservice.New()
    
    // Run concurrent operations
    var wg sync.WaitGroup
    errors := make(chan error, 10)
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            // Concurrent reads
            f, err := svc.Open("data")
            if err != nil {
                errors <- err
                return
            }
            defer f.Close()
            
            _, err = io.ReadAll(f)
            if err != nil {
                errors <- err
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    // Check for errors
    for err := range errors {
        t.Errorf("concurrent access error: %v", err)
    }
}
```

## Integration Testing

### Testing with VFS

```go
func TestServiceInVFS(t *testing.T) {
    // Create namespace
    ns := vfs.NewNamespace()
    
    // Mount service
    svc := myservice.New()
    err := ns.Mount(svc, "/svc")
    if err != nil {
        t.Fatal(err)
    }
    
    // Test through VFS
    f, err := ns.Open("/svc/hello")
    if err != nil {
        t.Fatal(err)
    }
    defer f.Close()
    
    // Verify functionality
    data, _ := io.ReadAll(f)
    if string(data) != "Hello, World!" {
        t.Errorf("unexpected content: %s", data)
    }
}
```

### Testing Task Integration

```go
func TestServiceWithTask(t *testing.T) {
    // Create task service
    taskSvc := task.NewService()
    
    // Create namespace with service
    ns := vfs.NewNamespace()
    ns.Mount(myservice.New(), "/svc")
    
    // Create task
    task := taskSvc.Create(task.Options{
        Namespace: ns,
        Command:   []string{"test"},
    })
    
    // Verify task can access service
    f, err := task.Namespace.Open("/svc/data")
    if err != nil {
        t.Errorf("task cannot access service: %v", err)
    }
    f.Close()
}
```

## Browser Testing

### WebAssembly Tests

For testing WASM-specific code:

```go
//go:build wasm

package web_test

import (
    "syscall/js"
    "testing"
)

func TestJSInterop(t *testing.T) {
    // Test JS global access
    global := js.Global()
    if global.IsUndefined() {
        t.Fatal("js.Global() is undefined")
    }
    
    // Test creating JS objects
    obj := js.ValueOf(map[string]interface{}{
        "test": true,
        "value": 42,
    })
    
    if !obj.Get("test").Bool() {
        t.Error("JS object property incorrect")
    }
}
```

### Manual Browser Testing

Create test pages for browser-specific features:

```html
<!-- test/browser/test.html -->
<!DOCTYPE html>
<html>
<head>
    <title>Wanix Browser Tests</title>
    <script src="/wanix.js"></script>
</head>
<body>
    <div id="results"></div>
    <script>
    async function runTests() {
        const results = document.getElementById('results');
        
        // Test 1: Initialize Wanix
        try {
            const wanix = await Wanix.init();
            results.innerHTML += '<p>✓ Wanix initialized</p>';
        } catch (e) {
            results.innerHTML += `<p>✗ Init failed: ${e}</p>`;
        }
        
        // Test 2: File operations
        try {
            await wanix.writeFile('/test.txt', 'Hello');
            const data = await wanix.readFile('/test.txt');
            if (data === 'Hello') {
                results.innerHTML += '<p>✓ File operations work</p>';
            } else {
                results.innerHTML += '<p>✗ File data mismatch</p>';
            }
        } catch (e) {
            results.innerHTML += `<p>✗ File ops failed: ${e}</p>`;
        }
    }
    
    runTests();
    </script>
</body>
</html>
```

## Performance Testing

### Benchmarking

```go
func BenchmarkFileRead(b *testing.B) {
    svc := myservice.New()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        f, _ := svc.Open("data")
        io.ReadAll(f)
        f.Close()
    }
}

func BenchmarkConcurrentRead(b *testing.B) {
    svc := myservice.New()
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            f, _ := svc.Open("data")
            io.ReadAll(f)
            f.Close()
        }
    })
}
```

### Memory Profiling

```go
func TestMemoryUsage(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping memory test in short mode")
    }
    
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    before := m.Alloc
    
    // Perform operations
    svc := myservice.New()
    for i := 0; i < 1000; i++ {
        f, _ := svc.Open("data")
        io.ReadAll(f)
        f.Close()
    }
    
    runtime.GC()
    runtime.ReadMemStats(&m)
    after := m.Alloc
    
    leaked := after - before
    if leaked > 1024*1024 { // 1MB threshold
        t.Errorf("possible memory leak: %d bytes", leaked)
    }
}
```

## Test Utilities

### Test Filesystem

```go
// testutil/fs.go
package testutil

import (
    "testing"
    "github.com/tractordev/wanix/fs/fskit"
)

func NewTestFS(t *testing.T) *fskit.MemFS {
    fs := fskit.NewMemFS()
    
    // Add test files
    must(fs.WriteFile("/test.txt", []byte("test data"), 0644))
    must(fs.Mkdir("/dir", 0755))
    must(fs.WriteFile("/dir/file.txt", []byte("nested"), 0644))
    
    return fs
}

func must(err error) {
    if err != nil {
        panic(err)
    }
}
```

### Test Namespace

```go
func NewTestNamespace(t *testing.T) *vfs.Namespace {
    ns := vfs.NewNamespace()
    
    // Mount test services
    ns.Mount(fskit.NewMemFS(), "/tmp")
    ns.Mount(testutil.NewTestFS(t), "/test")
    
    return ns
}
```

## Testing Best Practices

### 1. Table-Driven Tests

```go
func TestOperations(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"empty", "", "", true},
        {"valid", "test", "TEST", false},
        {"special", "!@#", "!@#", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := process(tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("process() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if result != tt.expected {
                t.Errorf("process() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### 2. Test Helpers

```go
func assertFileContent(t *testing.T, fs FS, path, want string) {
    t.Helper()
    
    data, err := fs.ReadFile(fs, path)
    if err != nil {
        t.Fatalf("ReadFile(%q) failed: %v", path, err)
    }
    
    if string(data) != want {
        t.Errorf("file %q content = %q, want %q", path, data, want)
    }
}
```

### 3. Cleanup

```go
func TestWithCleanup(t *testing.T) {
    // Create temporary resources
    tmpDir := t.TempDir() // Automatically cleaned up
    
    svc := myservice.New()
    t.Cleanup(func() {
        svc.Close() // Ensure cleanup
    })
    
    // Run tests...
}
```

### 4. Skip Conditions

```go
func TestBrowserOnly(t *testing.T) {
    if runtime.GOOS != "js" || runtime.GOARCH != "wasm" {
        t.Skip("test requires browser environment")
    }
    
    // Browser-specific tests...
}

func TestLongRunning(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping long test in short mode")
    }
    
    // Long-running tests...
}
```

## Continuous Integration

### GitHub Actions Test Workflow

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - uses: actions/setup-go@v4
      with:
        go-version: '1.23'
    
    - name: Run tests
      run: go test -v -race ./...
    
    - name: Generate coverage
      run: go test -coverprofile=coverage.out ./...
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

## Debugging Failed Tests

### Verbose Output

```bash
# Run with -v flag
go test -v ./fs/fskit

# Add custom logging
func TestDebug(t *testing.T) {
    t.Logf("Starting test with config: %+v", config)
    
    result := doOperation()
    t.Logf("Operation result: %+v", result)
}
```

### Test Isolation

```bash
# Run single test
go test -run TestSpecific ./package

# Run tests matching pattern
go test -run Test.*File ./...
```

### Race Detection

```bash
# Detect data races
go test -race ./...

# Fix race conditions found
```

## Next Steps

1. Write tests for all new features
2. Aim for >80% code coverage
3. Add benchmarks for performance-critical paths
4. Set up CI/CD with automated testing
5. Create browser test suite for web features