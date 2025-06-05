# Module System

The Wanix module system provides a pluggable architecture for extending the kernel with new file services and capabilities.

## Overview

Modules in Wanix are self-contained units that:
- Register file services into the namespace
- Provide new capabilities to processes
- Extend system functionality without kernel changes
- Can be loaded dynamically or compiled in

## Module Architecture

```
┌─────────────────────────────────────┐
│          Wanix Kernel               │
│     (Module Manager)                │
└──────────┬──────────────────────────┘
           │
    ┌──────▼──────┐
    │   Module    │
    │  Registry   │
    └──────┬──────┘
           │
    ┌──────┴──────┬────────┬──────────┐
    ▼             ▼        ▼          ▼
┌─────────┐ ┌─────────┐ ┌────────┐ ┌────────┐
│   Web   │ │   Cap   │ │  Task  │ │  VFS   │
│ Module  │ │ Module  │ │ Module │ │ Module │
└─────────┘ └─────────┘ └────────┘ └────────┘
```

## Core Components

### Module Interface (`wanix.go`)

Every module implements the Module interface:

```go
type Module interface {
    Name() string
    Init(kernel *Kernel) error
    Mount(ns *vfs.Namespace) error
    Shutdown() error
}
```

### Module Registration

Modules register during kernel initialization:

```go
// In wanix.go
func (k *Kernel) RegisterModule(mod Module) error {
    k.modules[mod.Name()] = mod
    return mod.Init(k)
}

// During startup
kernel.RegisterModule(web.NewModule())
kernel.RegisterModule(cap.NewModule())
kernel.RegisterModule(task.NewModule())
```

## Built-in Modules

### 1. Web Module (`web/web.go`)

Provides browser API access:
- DOM manipulation (`/web/dom`)
- Web Workers (`/web/worker`)
- Service Workers (`/web/sw`)
- File System Access (`/web/fsa`)

```go
type WebModule struct {
    dom    *dom.Service
    worker *worker.Service
    sw     *sw.Service
}

func (m *WebModule) Mount(ns *vfs.Namespace) error {
    ns.Mount(m.dom, "/web/dom")
    ns.Mount(m.worker, "/web/worker")
    ns.Mount(m.sw, "/web/sw")
    return nil
}
```

### 2. Capability Module (`cap/service.go`)

Manages system capabilities:
- Resource allocation (`/cap/new/*`)
- Capability lifecycle
- Access control

```go
type CapModule struct {
    registry map[string]CapabilityFactory
    active   map[string]Capability
}

func (m *CapModule) Mount(ns *vfs.Namespace) error {
    ns.Mount(m, "/cap")
    return nil
}
```

### 3. Task Module (`task/service.go`)

Process management:
- Task creation (`/task/new`)
- Process information (`/task/$id/*`)
- Resource accounting

```go
type TaskModule struct {
    tasks map[string]*Task
    mu    sync.RWMutex
}

func (m *TaskModule) Mount(ns *vfs.Namespace) error {
    ns.Mount(m, "/task")
    return nil
}
```

### 4. VFS Module (`vfs/vfs.go`)

Core namespace management:
- Mount operations
- Bind operations
- Union filesystem support

## Creating Custom Modules

### Basic Module Structure

```go
package mymodule

import (
    "github.com/tractordev/wanix"
    "github.com/tractordev/wanix/vfs"
)

type MyModule struct {
    kernel *wanix.Kernel
    data   map[string]interface{}
}

func NewModule() wanix.Module {
    return &MyModule{
        data: make(map[string]interface{}),
    }
}

func (m *MyModule) Name() string {
    return "mymodule"
}

func (m *MyModule) Init(kernel *wanix.Kernel) error {
    m.kernel = kernel
    // Initialize module resources
    return nil
}

func (m *MyModule) Mount(ns *vfs.Namespace) error {
    // Mount module file services
    ns.Mount(m, "/mymodule")
    return nil
}

func (m *MyModule) Shutdown() error {
    // Cleanup resources
    return nil
}
```

### Implementing File Service

```go
// Implement fs.FS interface
func (m *MyModule) Open(name string) (fs.File, error) {
    // Route to appropriate handler
    switch name {
    case "ctl":
        return m.openControl()
    case "status":
        return m.openStatus()
    default:
        return nil, fs.ErrNotExist
    }
}

func (m *MyModule) openControl() (fs.File, error) {
    return fskit.NewControlFile(m, []Command{
        {"start", "Start the service", m.cmdStart},
        {"stop", "Stop the service", m.cmdStop},
    }), nil
}
```

## Module Communication

### 1. Direct Kernel Access

Modules can access kernel services:

```go
func (m *MyModule) needsTask() error {
    taskSvc := m.kernel.GetModule("task").(*task.Service)
    newTask := taskSvc.CreateTask()
    // Configure and start task
}
```

### 2. File-based Communication

Modules communicate through filesystem:

```go
// Module A exposes data
ns.Mount(dataFS, "/moduleA/data")

// Module B reads it
data, _ := fs.ReadFile(ns, "/moduleA/data/info")
```

### 3. Event System

Modules can subscribe to kernel events:

```go
func (m *MyModule) Init(kernel *wanix.Kernel) error {
    kernel.Subscribe("task.exit", m.onTaskExit)
    return nil
}

func (m *MyModule) onTaskExit(e Event) {
    // Handle task exit event
}
```

## Module Lifecycle

### 1. Loading Phase
```
Kernel Start
    ↓
RegisterModule()
    ↓
Module.Init()
    ↓
Module added to registry
```

### 2. Mounting Phase
```
Process Creation
    ↓
Namespace Creation
    ↓
Module.Mount() for each module
    ↓
Module services available in namespace
```

### 3. Runtime Phase
- Modules handle file operations
- Manage their resources
- Communicate with other modules

### 4. Shutdown Phase
```
Kernel Shutdown
    ↓
Module.Shutdown() for each module
    ↓
Resources cleaned up
```

## Advanced Module Features

### 1. Lazy Loading

Modules can be loaded on-demand:

```go
func (k *Kernel) LoadModule(name string) error {
    if _, exists := k.modules[name]; exists {
        return nil
    }
    
    mod := loadModuleByName(name)
    return k.RegisterModule(mod)
}
```

### 2. Module Dependencies

Declare and check dependencies:

```go
func (m *MyModule) Dependencies() []string {
    return []string{"vfs", "task"}
}

func (m *MyModule) Init(kernel *wanix.Kernel) error {
    // Check dependencies
    for _, dep := range m.Dependencies() {
        if !kernel.HasModule(dep) {
            return fmt.Errorf("missing dependency: %s", dep)
        }
    }
    return nil
}
```

### 3. Hot Reload

Some modules support hot reload:

```go
func (m *MyModule) Reload() error {
    // Save state
    state := m.saveState()
    
    // Reinitialize
    m.Shutdown()
    m.Init(m.kernel)
    
    // Restore state
    m.restoreState(state)
    
    return nil
}
```

## Security Considerations

### 1. Module Isolation
- Modules run in kernel space (trusted)
- Must validate all user input
- Should minimize shared state

### 2. Capability Control
- Modules define their own capabilities
- Must enforce access control
- Should use principle of least privilege

### 3. Resource Management
- Track and limit resource usage
- Clean up on shutdown
- Handle errors gracefully

## Best Practices

1. **Single Responsibility**: Each module should do one thing well
2. **File-based API**: Expose functionality through files
3. **Stateless Design**: Minimize module state when possible
4. **Error Handling**: Always return meaningful errors
5. **Documentation**: Document control commands and file formats

## Example: Timer Module

Here's a complete example of a timer module:

```go
package timer

type TimerModule struct {
    timers map[string]*Timer
    mu     sync.RWMutex
}

func (m *TimerModule) Mount(ns *vfs.Namespace) error {
    ns.Mount(m, "/dev/timer")
    return nil
}

func (m *TimerModule) Open(name string) (fs.File, error) {
    if name == "new" {
        return m.allocateTimer()
    }
    
    // Parse timer ID from path
    timerID := strings.TrimPrefix(name, "timer/")
    
    m.mu.RLock()
    timer, exists := m.timers[timerID]
    m.mu.RUnlock()
    
    if !exists {
        return nil, fs.ErrNotExist
    }
    
    return timer.Open(name)
}

func (m *TimerModule) allocateTimer() (fs.File, error) {
    timer := &Timer{
        id: generateID(),
        ch: make(chan time.Time),
    }
    
    m.mu.Lock()
    m.timers[timer.id] = timer
    m.mu.Unlock()
    
    return fskit.NewReadFile([]byte(timer.id)), nil
}
```

This modular architecture makes Wanix highly extensible while maintaining clean interfaces and separation of concerns.