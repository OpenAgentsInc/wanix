//go:build js && wasm

package vm

import (
	"context"
	"io"
	"log"
	"syscall/js"

	"tractor.dev/toolkit-go/engine/cli"
	"tractor.dev/wanix/fs"
	"tractor.dev/wanix/fs/fskit"
	"tractor.dev/wanix/internal"
)

type VM struct {
	id     int
	typ    string
	value  js.Value
	serial *serial
}

func (r *VM) Value() js.Value {
	return r.value
}

func (r *VM) Open(name string) (fs.File, error) {
	return r.OpenContext(context.Background(), name)
}

func (r *VM) ResolveFS(ctx context.Context, name string) (fs.FS, string, error) {
	fsys := fskit.MapFS{
		"ctl": internal.ControlFile(r.makeCtlCommand()),
		"type": internal.FieldFile(r.typ),
	}
	// Note: ttyS0 is not included here because it will be bound from outside
	return fs.Resolve(fsys, ctx, name)
}

func (r *VM) OpenContext(ctx context.Context, name string) (fs.File, error) {
	fsys, rname, err := r.ResolveFS(ctx, name)
	if err != nil {
		return nil, err
	}
	return fs.OpenContext(ctx, fsys, rname)
}


func (r *VM) makeCtlCommand() *cli.Command {
	return &cli.Command{
		Usage: "ctl",
		Short: "control the resource",
		Run: func(ctx *cli.Context, args []string) {
			switch args[0] {
			case "start":
				// Get the filesystem from the command context
				// This gives us access to the task's namespace where ttyS0 is bound
				fsys, _, ok := fs.Origin(ctx.Context)
				if ok {
					// Try to open ttyS0 from the task's namespace
					if tty, err := fsys.Open("ttyS0"); err == nil {
						log.Println("vm start: connected to ttyS0")
						go io.Copy(r.serial, tty)
						if w, ok := tty.(io.Writer); ok {
							go io.Copy(w, r.serial)
						}
					} else {
						log.Printf("vm start: ttyS0 not available: %v", err)
					}
				} else {
					log.Println("vm start: no filesystem context available")
				}
				
				// Start the VM
				r.value.Get("ready").Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
					r.value.Call("run")
					return nil
				}))
			}
		},
	}
}

type serial struct {
	js.Value
	buf *internal.BufferedPipe
}

func newSerial(vm js.Value) *serial {
	buf := internal.NewBufferedPipe(true)
	vm.Call("add_listener", "serial0-output-byte", js.FuncOf(func(this js.Value, args []js.Value) any {
		buf.Write([]byte{byte(args[0].Int())})
		return nil
	}))
	return &serial{
		Value: vm,
		buf:   buf,
	}
}

func (s *serial) Write(p []byte) (n int, err error) {
	buf := js.Global().Get("Uint8Array").New(len(p))
	n = js.CopyBytesToJS(buf, p)
	s.Value.Call("serial_send_bytes", 0, buf)
	return
}

func (s *serial) Read(p []byte) (int, error) {
	return s.buf.Read(p)
}
