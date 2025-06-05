//go:build js && wasm

package node

import (
	"context"

	"tractor.dev/wanix/fs"
	"tractor.dev/wanix/fs/fskit"
)

const bootstrapJS = `(function(global) {
    // Minimal console implementation
    global.console = {
        log: function(...args) {
            const output = args.map(arg => {
                if (typeof arg === 'object') {
                    try {
                        return JSON.stringify(arg);
                    } catch (e) {
                        return '[object Object]';
                    }
                }
                return String(arg);
            }).join(' ') + '\n';
            // Write to Wanix stdout (fd 1)
            __wanix_write(1, output);
        },
        error: function(...args) {
            const output = args.map(arg => {
                if (typeof arg === 'object') {
                    try {
                        return JSON.stringify(arg);
                    } catch (e) {
                        return '[object Object]';
                    }
                }
                return String(arg);
            }).join(' ') + '\n';
            // Write to Wanix stderr (fd 2)
            __wanix_write(2, output);
        }
    };
    
    // Minimal process object
    global.process = {
        stdout: {
            write: function(data) {
                __wanix_write(1, String(data));
            }
        },
        stderr: {
            write: function(data) {
                __wanix_write(2, String(data));
            }
        },
        exit: function(code) {
            __wanix_exit(code || 0);
        },
        argv: ['node'],  // Will be populated later
        env: {},         // Will be populated later
        cwd: function() {
            return '/';  // For now, hardcode root
        }
    };
    
    // Minimal require (just returns empty object for now)
    global.require = function(id) {
        return {};
    };
    
    // Add global to itself
    global.global = global;
})(this);`

type Service struct {
	// Simple service with no state for now
}

func New() *Service {
	return &Service{}
}

func (s *Service) Open(name string) (fs.File, error) {
	return s.OpenContext(context.Background(), name)
}

func (s *Service) OpenContext(ctx context.Context, name string) (fs.File, error) {
	fsys := fskit.MapFS{
		"bootstrap.js": fskit.Entry("bootstrap.js", 0444, []byte(bootstrapJS)),
		"ctl": fskit.Entry("ctl", 0644, []byte{}),
	}
	return fs.OpenContext(ctx, fsys, name)
}

func GetBootstrapJS() string {
	return bootstrapJS
}