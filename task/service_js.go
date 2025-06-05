//go:build js && wasm

package task

import "log"

func registerPlatformTasks(d *Service) {
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
}