//go:build !js || !wasm

package task

func registerPlatformTasks(d *Service) {
	// No platform-specific tasks for non-WebAssembly builds
}