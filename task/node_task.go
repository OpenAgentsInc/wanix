//go:build js && wasm

package task

import (
	"context"
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

	// Create a new JavaScript context with global object
	// For MVP, we use js.Global() as the context
	global := js.Global()

	// Set up Wanix functions before loading bootstrap
	global.Set("__wanix_write", js.FuncOf(func(this js.Value, args []js.Value) any {
		fd := args[0].Int()
		output := args[1].String()

		if fdFS, ok := t.res.fds[strconv.Itoa(fd)]; ok {
			fdFile, err := fdFS.Open(".")
			if err != nil {
				log.Printf("nodejs task: error opening fd %d: %v", fd, err)
				return nil
			}
			defer fdFile.Close()

			if writer, ok := fdFile.(io.Writer); ok {
				_, err = writer.Write([]byte(output))
				if err != nil {
					log.Printf("nodejs task: error writing to fd %d: %v", fd, err)
				}
			} else {
				log.Printf("nodejs task: fd %d is not writable", fd)
			}
		} else {
			log.Printf("nodejs task: invalid fd %d", fd)
		}
		return nil
	}))

	global.Set("__wanix_exit", js.FuncOf(func(this js.Value, args []js.Value) any {
		code := 0
		if len(args) > 0 {
			code = args[0].Int()
		}
		t.res.exit = strconv.Itoa(code)
		return nil
	}))

	// Load bootstrap.js to set up Node.js globals
	bootstrapFile, err := t.res.ns.OpenContext(context.Background(), "/web/node/bootstrap.js")
	if err != nil {
		return fmt.Errorf("nodejs task: failed to open bootstrap.js: %v", err)
	}
	defer bootstrapFile.Close()

	bootstrapData, err := io.ReadAll(bootstrapFile)
	if err != nil {
		return fmt.Errorf("nodejs task: failed to read bootstrap.js: %v", err)
	}

	// Execute bootstrap to set up Node.js environment
	global.Call("eval", string(bootstrapData))

	// Now execute the user script
	global.Call("eval", script)

	// If process.exit() was not called, we assume success.
	if t.res.exit == "" {
		t.res.exit = "0"
	}

	log.Println("Nodejs task finished.")
	return nil
}