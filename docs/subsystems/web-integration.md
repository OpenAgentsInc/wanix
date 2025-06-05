# Web Integration Subsystem

The web integration subsystem exposes browser APIs as file services, enabling Wanix to leverage web platform features through its unified filesystem interface.

## Overview

Web integration provides file-based access to:
- **DOM manipulation** - Create and control page elements
- **Web Workers** - Background JavaScript execution
- **Service Workers** - Network request interception
- **File System Access** - Native file picker integration
- **WebSockets** - Network communication
- **OPFS** - Browser storage

## Architecture

```
/web/
├── dom/          # DOM manipulation
│   ├── body/     # Document body
│   ├── new/      # Element creation
│   └── <id>/     # Element instances
├── worker/       # Web Workers
│   ├── new       # Worker creation
│   └── <id>/     # Worker instances
├── sw/           # Service Worker
│   ├── ctl       # Control file
│   └── routes    # URL routing
├── opfs/         # Origin Private FS
└── ws/           # WebSockets
    ├── new       # Connection creation
    └── <id>/     # Connections
```

## DOM Service (`web/dom/service.go`)

### Architecture

The DOM service provides file-based manipulation of web page elements:

```go
type Service struct {
    elements map[string]Element
    document js.Value
    mu       sync.RWMutex
}

type Element interface {
    fs.FS
    Type() string
    JSValue() js.Value
}
```

### Element Creation

Elements are created through the clone pattern:

```bash
# Create an iframe
id=$(cat /web/dom/new/iframe)

# Configure attributes
echo "src=/index.html" >> /web/dom/$id/attrs
echo "width=800" >> /web/dom/$id/attrs

# Add to page
echo "append-child $id" > /web/dom/body/ctl
```

Implementation:
```go
func (s *Service) createElement(elemType string) (string, error) {
    // Create DOM element
    elem := js.Global().Get("document").Call("createElement", elemType)
    
    // Wrap in element interface
    id := generateID()
    s.elements[id] = &DOMElement{
        id:   id,
        elem: elem,
        typ:  elemType,
    }
    
    return id, nil
}
```

### Element Interface

Each element exposes:
```
/web/dom/<id>/
├── ctl        # Control commands
├── attrs      # Element attributes
├── style      # CSS styles
├── class      # CSS classes
├── inner      # innerHTML
└── data       # Special data (for terminals)
```

### Special Elements

#### XTerm Terminal (`web/dom/xterm/`)

```go
type XTermElement struct {
    *DOMElement
    terminal js.Value
    onData   js.Func
}

func (x *XTermElement) Open(name string) (fs.File, error) {
    if name == "data" {
        // Return duplex stream for terminal I/O
        return x.dataFile, nil
    }
    return x.DOMElement.Open(name)
}
```

#### Style Element

Append CSS to the page:
```bash
echo "body { background: #000; }" >> /web/dom/style
```

### Control Commands

Common DOM control commands:

```go
var domCommands = []Command{
    {"append-child", appendChildCmd},
    {"remove-child", removeChildCmd},
    {"remove", removeCmd},
    {"focus", focusCmd},
    {"click", clickCmd},
}

func appendChildCmd(elem *DOMElement, args []string) error {
    childID := args[0]
    child := getElement(childID)
    elem.JSValue().Call("appendChild", child.JSValue())
    return nil
}
```

## Worker Service (`web/worker/service.go`)

### Web Worker Management

```go
type WorkerService struct {
    workers map[string]*Worker
    mu      sync.RWMutex
}

type Worker struct {
    id       string
    worker   js.Value
    status   string
    messages chan Message
}
```

### Worker Lifecycle

```bash
# Create worker
id=$(cat /web/worker/new)

# Start with script
echo "start /scripts/worker.js" > /web/worker/$id/ctl

# Send message
echo '{"cmd": "process", "data": [1,2,3]}' > /web/worker/$id/send

# Read messages
cat /web/worker/$id/recv

# Terminate
echo "terminate" > /web/worker/$id/ctl
```

### Worker Implementation

```go
func (w *Worker) Start(script string) error {
    // Create web worker
    w.worker = js.Global().Get("Worker").New(script)
    
    // Set up message handling
    w.worker.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        msg := args[0].Get("data")
        w.messages <- Message{Data: msg}
        return nil
    }))
    
    w.status = "running"
    return nil
}
```

## Service Worker (`web/sw/service.go`)

### Service Worker Control

The service worker enables:
- Offline functionality
- URL routing to namespace
- Caching strategies

```go
type ServiceWorker struct {
    registration js.Value
    routes       map[string]Route
}

type Route struct {
    Pattern string
    Target  string
    Cache   bool
}
```

### URL Routing

Map URLs to namespace paths:

```bash
# Route /app/* to namespace
echo "route /app/* /web/opfs/app" > /web/sw/routes

# Enable caching
echo "cache /static/*" > /web/sw/cache
```

### Implementation

```javascript
// In service worker
self.addEventListener('fetch', (event) => {
    const url = new URL(event.request.url);
    
    // Check if URL matches namespace route
    if (url.pathname.startsWith('/:/')) {
        // Route to Wanix namespace
        const nsPath = url.pathname.substring(3);
        event.respondWith(handleNamespaceRequest(nsPath));
    }
});
```

## OPFS Integration (`web/fsa/fs.go`)

### Origin Private File System

Direct access to browser storage:

```go
type OPFS struct {
    root js.Value  // FileSystemDirectoryHandle
}

func (o *OPFS) Open(name string) (fs.File, error) {
    parts := strings.Split(name, "/")
    handle := o.root
    
    // Navigate to file
    for _, part := range parts[:len(parts)-1] {
        handle = await(handle.Call("getDirectoryHandle", part))
    }
    
    // Get file handle
    fileHandle := await(handle.Call("getFileHandle", parts[len(parts)-1]))
    
    return &opfsFile{handle: fileHandle}, nil
}
```

### File Operations

```go
type opfsFile struct {
    handle   js.Value
    writable js.Value
}

func (f *opfsFile) Write(p []byte) (int, error) {
    if f.writable.IsUndefined() {
        f.writable = await(f.handle.Call("createWritable"))
    }
    
    // Write data
    buf := js.Global().Get("Uint8Array").New(len(p))
    js.CopyBytesToJS(buf, p)
    await(f.writable.Call("write", buf))
    
    return len(p), nil
}
```

## File System Access API (`web/fsa/fsa.go`)

### Native File Picker

```go
type PickerFS struct {
    handle js.Value
}

func (p *PickerFS) ShowPicker() error {
    // Show directory picker
    promise := js.Global().Call("showDirectoryPicker", map[string]interface{}{
        "mode": "readwrite",
    })
    
    p.handle = await(promise)
    return nil
}
```

### Permission Handling

```go
func (p *PickerFS) checkPermission() error {
    perm := await(p.handle.Call("queryPermission", map[string]interface{}{
        "mode": "readwrite",
    }))
    
    if perm.String() != "granted" {
        // Request permission
        perm = await(p.handle.Call("requestPermission", map[string]interface{}{
            "mode": "readwrite",
        }))
    }
    
    if perm.String() != "granted" {
        return errors.New("permission denied")
    }
    return nil
}
```

## WebSocket Service (`web/ws/`)

### WebSocket Connections

```go
type WebSocket struct {
    id     string
    conn   js.Value
    recv   chan []byte
    send   chan []byte
    status string
}

func (w *WebSocket) Connect(url string) error {
    // Create WebSocket
    w.conn = js.Global().Get("WebSocket").New(url)
    
    // Set up handlers
    w.conn.Set("onmessage", js.FuncOf(w.onMessage))
    w.conn.Set("onclose", js.FuncOf(w.onClose))
    w.conn.Set("onerror", js.FuncOf(w.onError))
    
    // Wait for connection
    return w.waitForOpen()
}
```

### File Interface

```bash
# Create WebSocket
id=$(cat /web/ws/new)

# Connect
echo "connect wss://example.com/socket" > /web/ws/$id/ctl

# Send message
echo "Hello, server!" > /web/ws/$id/send

# Read messages
cat /web/ws/$id/recv

# Close connection
echo "close" > /web/ws/$id/ctl
```

## JavaScript Bridge (`web/jsutil/`)

### Utilities for Go/JS Interaction

```go
// Await promises
func Await(promise js.Value) (js.Value, error) {
    done := make(chan struct{})
    var result js.Value
    var err error
    
    promise.Call("then",
        js.FuncOf(func(this js.Value, args []js.Value) interface{} {
            result = args[0]
            close(done)
            return nil
        }),
        js.FuncOf(func(this js.Value, args []js.Value) interface{} {
            err = js.Error{args[0]}
            close(done)
            return nil
        }),
    )
    
    <-done
    return result, err
}

// Load JavaScript modules
func LoadModule(url string) (js.Value, error) {
    promise := js.Global().Call("import", url)
    return Await(promise)
}
```

## Integration Patterns

### 1. Event Handling

```go
// File-based event streams
type EventFile struct {
    events chan Event
}

func (e *EventFile) Read(p []byte) (int, error) {
    event := <-e.events
    data, _ := json.Marshal(event)
    return copy(p, data), nil
}
```

### 2. Streaming Data

```go
// Bidirectional streaming
type StreamFile struct {
    read  io.Reader
    write io.Writer
}

func (s *StreamFile) Read(p []byte) (int, error) {
    return s.read.Read(p)
}

func (s *StreamFile) Write(p []byte) (int, error) {
    return s.write.Write(p)
}
```

### 3. Async Operations

```go
// Async file operations
type AsyncFile struct {
    pending []js.Value
}

func (a *AsyncFile) Sync() error {
    // Wait for all pending operations
    promises := js.Global().Get("Promise").Call("all", a.pending)
    _, err := Await(promises)
    return err
}
```

## Security Considerations

### 1. Input Validation

```go
func validateElementType(typ string) error {
    allowed := []string{"div", "span", "iframe", "img", "video"}
    for _, a := range allowed {
        if typ == a {
            return nil
        }
    }
    return errors.New("element type not allowed")
}
```

### 2. CSP Compliance

```go
// Respect Content Security Policy
func checkCSP(action string) bool {
    csp := js.Global().Get("document").Get("contentSecurityPolicy")
    // Check if action is allowed
    return true
}
```

### 3. Origin Isolation

```go
// Ensure same-origin policy
func checkOrigin(url string) error {
    current := js.Global().Get("location").Get("origin").String()
    parsed, _ := url.Parse(url)
    if parsed.Host != current {
        return errors.New("cross-origin access denied")
    }
    return nil
}
```

## Performance Tips

1. **Batch DOM Operations**
```go
// Queue operations and flush together
type DOMBatch struct {
    ops []func()
}

func (b *DOMBatch) Flush() {
    js.Global().Call("requestAnimationFrame", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        for _, op := range b.ops {
            op()
        }
        return nil
    }))
}
```

2. **Efficient Data Transfer**
```go
// Use transferable objects
func transferArrayBuffer(data []byte) js.Value {
    buf := js.Global().Get("ArrayBuffer").New(len(data))
    js.CopyBytesToJS(buf, data)
    return buf
}
```

3. **Resource Management**
```go
// Clean up JS callbacks
func (e *Element) Close() error {
    // Release JS functions
    for _, fn := range e.callbacks {
        fn.Release()
    }
    return nil
}
```

## Future Enhancements

- **WebRTC Support**: P2P communication through files
- **WebGL/WebGPU**: Graphics through file interfaces
- **Media Streams**: Camera/microphone access
- **Bluetooth/USB**: Hardware device access
- **Notifications**: System notifications as files