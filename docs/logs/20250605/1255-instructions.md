Hello! As your coding agent, I will now begin implementing the MVP for Node.js support in Wanix. I'll start with the first set of tasks from your issue description, focusing on the core runtime components.

My plan is as follows:

1.  **Create `task/node_task.go`**: This new file will define the `NodeTask` type, which is responsible for executing JavaScript code within the Wanix task system. It will include the logic for setting up a basic JavaScript context and wiring up `stdout`/`stderr`.
2.  **Modify `task/service.go`**: I will update the existing task service to recognize and initialize the new "nodejs" task type.

Let's start with the first step.

**Step 1: Create `task/node_task.go`**

I will create a new file `task/node_task.go` to house the logic for our Node.js task runner. This component will handle the execution of JavaScript code, including setting up the necessary JS environment and functions to communicate with Wanix's file system for I/O.

```go
// Create a new file named `task/node_task.go` with the following content:
package task

import (
	"fmt"
	"io"
	"log"
	"strconv"
	"syscall/js"
)

// NodeTask represents a task that executes JavaScript code in a Node.js-like environment.
type NodeTask struct {
	res *Resource
}

// NewNodeTask creates a new NodeTask.
func NewNodeTask(res *Resource) *NodeTask {
	return &NodeTask{res: res}
}

// Start executes the JavaScript module associated with the task.
func (t *NodeTask) Start() error {
	log.Println("Starting nodejs task...")

	// The script content is expected to be in the Cmd field.
	script := t.res.Cmd()
	if script == "" {
		return fmt.Errorf("nodejs task: no script provided in cmd field")
	}

	// TODO: For real isolation, this should use a new Worker or a sandboxed iframe.
	// For the MVP, we use an IIFE to create a new scope, which is not true isolation.
	runtime := js.Global().Call("eval", `(function() {
		const exports = {};
		const module = { exports };
		return { module, exports };
	})()`)

	// __wanix_write writes to the task's file descriptors (stdout, stderr).
	// This will be called by the JS bootstrap code for console.log etc.
	runtime.Set("__wanix_write", js.FuncOf(func(this js.Value, args []js.Value) any {
		fd := args[0].Int()
		output := args[1].String()

		var writer io.Writer
		if f, ok := t.res.fds[strconv.Itoa(fd)]; ok {
			var err error
			writer, err = f.Open(".")
			if err != nil {
				log.Printf("nodejs task: error opening fd %d: %v", fd, err)
				return nil
			}
		} else {
			log.Printf("nodejs task: invalid fd %d", fd)
			return nil
		}

		_, err := writer.Write([]byte(output))
		if err != nil {
			log.Printf("nodejs task: error writing to fd %d: %v", fd, err)
		}
		return nil
	}))

	// __wanix_exit terminates the task with a given exit code.
	// This will be called by `process.exit()`.
	runtime.Set("__wanix_exit", js.FuncOf(func(this js.Value, args []js.Value) any {
		code := 0
		if len(args) > 0 {
			code = args[0].Int()
		}
		t.res.exit = strconv.Itoa(code)
		return nil
	}))

	// Execute the script. The script is expected to be wrapped in a function.
	// We will later load a bootstrap script first that defines globals like `console`.
	// For now, this is a placeholder for where that logic will go.
	// A simple eval is enough for the "Core Runtime" part of the MVP.
	runtime.Call("eval", script)

	// If process.exit() was not called, we assume success.
	if t.res.exit == "" {
		t.res.exit = "0"
	}

	log.Println("Nodejs task finished.")
	return nil
}
```

**Step 2: Modify `task/service.go`**

Next, I'll register the new `nodejs` task type within the main task service. This allows the Wanix system to create and manage `NodeTask` instances.

```go
// In the file `task/service.go`, add the registration for the "nodejs" task type.

// At the top of the file, add the following import if it's not already there:
import (
	"log"
)

// In the `New()` function, after the existing `d.Register("ns", ...)` call,
// add the following code to register the "nodejs" task type.
// This should be around line 30.

	d.Register("nodejs", func(p *Resource) error {
		nodeTask := NewNodeTask(p)
		go func() {
			if err := nodeTask.Start(); err != nil {
				log.Printf("nodejs task error: %v", err)
				p.exit = "1"
			}
		}()
		return nil
	})

// The final `New` function in `task/service.go` should look like this:
func New() *Service {
	d := &Service{
		types:     make(map[string]func(*Resource) error),
		resources: make(map[string]fs.FS),
		nextID:    0,
	}
	// empty namespace process
	d.Register("ns", func(_ *Resource) error {
		return nil
	})
	d.Register("nodejs", func(p *Resource) error {
		nodeTask := NewNodeTask(p)
		go func() {
			if err := nodeTask.Start(); err != nil {
				log.Printf("nodejs task error: %v", err)
				p.exit = "1"
			}
		}()
		return nil
	})
	return d
}
```

This completes the initial setup for the core Node.js runtime. We now have a `NodeTask` type and the system knows how to create it. The next steps will build on this foundation by adding the minimal module system and the `/bin/node` executable.

I will proceed with the next set of tasks in my following response.
