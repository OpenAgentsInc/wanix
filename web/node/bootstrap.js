(function(global) {
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
})(this);