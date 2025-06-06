//go:build js && wasm

package task

import "log"

func registerPlatformTasks(d *Service) {
	// Register the nodejs task type
	d.Register("nodejs", func(p *Resource) error {
		// This function is the starter - called when "start" is written to ctl
		log.Printf("nodejs starter called, cmd length: %d", len(p.Cmd()))
		
		nodeTask := NewNodeTask(p)
		go func() {
			log.Println("Starting NodeTask.Start() in goroutine")
			if err := nodeTask.Start(); err != nil {
				log.Printf("nodejs task error: %v", err)
				p.exit = "1"
			}
			log.Println("NodeTask.Start() completed")
		}()
		return nil
	})
	
	// Register a test nodejs task that sets a hardcoded script
	d.Register("nodejs-test", func(p *Resource) error {
		log.Println("Created nodejs-test task with hardcoded script")
		// Set the command but don't start immediately
		p.cmd = `
console.log("=== Node.js Test Task ===");
console.log("This is a hardcoded test script.");
console.log("If you see this, Node.js support is working!");
console.error("This is stderr output");
console.log("Process info:", { cwd: process.cwd(), argv: process.argv });
console.log("=== Test Complete ===");`
		
		// Don't start here - wait for "start" command
		return nil
	})
}