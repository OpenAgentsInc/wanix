package task

import (
	"context"
	"log"
	"strconv"
	"strings"

	"tractor.dev/wanix/fs"
	"tractor.dev/wanix/fs/fskit"
	"tractor.dev/wanix/internal"
	"tractor.dev/wanix/vfs"
)

type Service struct {
	types     map[string]func(*Resource) error
	resources map[string]fs.FS
	nextID    int
}

// Ensure Service implements the required interfaces
var _ fs.FS = (*Service)(nil)
var _ fs.CreateFS = (*Service)(nil)
var _ fs.ResolveFS = (*Service)(nil)

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
	// Platform-specific registrations are in service_*.go files
	registerPlatformTasks(d)
	return d
}

func (d *Service) Register(kind string, starter func(*Resource) error) {
	d.types[kind] = starter
}

func (d *Service) Alloc(kind string, parent *Resource) (*Resource, error) {
	starter, ok := d.types[kind]
	if !ok {
		return nil, fs.ErrNotExist
	}
	d.nextID++
	rid := strconv.Itoa(d.nextID)

	a0, b0 := internal.BufferedConnPipe(false)
	a1, b1 := internal.BufferedConnPipe(false)
	a2, b2 := internal.BufferedConnPipe(false)

	p := &Resource{
		starter: starter,
		id:      d.nextID,
		typ:     kind,
		fds: map[string]fs.FS{
			"0": newFdFile(a0, "0"),
			"1": newFdFile(a1, "1"),
			"2": newFdFile(a2, "2"),
		},
		sys: map[string]fs.FS{
			"fd0": newFdFile(b0, "fd0"),
			"fd1": newFdFile(b1, "fd1"),
			"fd2": newFdFile(b2, "fd2"),
		},
	}
	ctx := context.WithValue(context.Background(), TaskContextKey, p)
	if parent != nil {
		p.parent = parent
		p.ns = parent.ns.Clone(ctx)
	} else {
		p.ns = vfs.New(ctx)
	}
	d.resources[rid] = p
	log.Printf("Task.Alloc: created task %s (type=%s, id=%d)", rid, kind, d.nextID)
	return p, nil
}

func (d *Service) ResolveFS(ctx context.Context, name string) (fs.FS, string, error) {
	// Special case: if resolving ".", return the service itself
	if name == "." {
		return d, ".", nil
	}
	
	m := fskit.MapFS{
		"new": fskit.OpenFunc(func(ctx context.Context, name string) (fs.File, error) {
			if name == "." {
				var nodes []fs.DirEntry
				for kind := range d.types {
					nodes = append(nodes, fskit.Entry(kind, 0555))
				}
				return fskit.DirFile(fskit.Entry("new", 0555), nodes...), nil
			}
			return &fskit.FuncFile{
				Node: fskit.Entry(name, 0555),
				ReadFunc: func(n *fskit.Node) error {
					t, _ := FromContext(ctx)
					p, err := d.Alloc(name, t)
					if err != nil {
						return err
					}
					fskit.SetData(n, []byte(p.ID()+"\n"))
					return nil
				},
			}, nil
		}),
	}
	t, ok := FromContext(ctx)
	if ok {
		m["self"] = internal.FieldFile(t.ID(), nil)
	}
	return fskit.UnionFS{m, fskit.MapFS(d.resources)}, name, nil
}

func (d *Service) Stat(name string) (fs.FileInfo, error) {
	log.Println("bare stat:", name)
	return d.StatContext(context.Background(), name)
}

func (d *Service) StatContext(ctx context.Context, name string) (fs.FileInfo, error) {
	// Handle root directory specially to avoid infinite recursion
	if name == "." {
		return fskit.Entry(".", fs.ModeDir|0755), nil
	}
	
	fsys, rname, err := d.ResolveFS(ctx, name)
	if err != nil {
		return nil, err
	}
	return fs.StatContext(ctx, fsys, rname)
}

func (d *Service) Open(name string) (fs.File, error) {
	log.Println("bare open:", name)
	return d.OpenContext(context.Background(), name)
}

func (d *Service) OpenContext(ctx context.Context, name string) (fs.File, error) {
	// Handle root directory specially to avoid infinite recursion
	if name == "." {
		var entries []fs.DirEntry
		// Add "new" directory
		entries = append(entries, fskit.Entry("new", fs.ModeDir|0755))
		// Add task IDs
		for id := range d.resources {
			entries = append(entries, fskit.Entry(id, fs.ModeDir|0755))
		}
		// Add "self" if in task context
		if _, ok := FromContext(ctx); ok {
			entries = append(entries, fskit.Entry("self", 0444))
		}
		return fskit.DirFile(fskit.Entry(".", fs.ModeDir|0755), entries...), nil
	}
	
	fsys, rname, err := d.ResolveFS(ctx, name)
	if err != nil {
		return nil, err
	}
	return fs.OpenContext(ctx, fsys, rname)
}

func (d *Service) Create(name string) (fs.File, error) {
	return d.CreateContext(context.Background(), name)
}

func (d *Service) CreateContext(ctx context.Context, name string) (fs.File, error) {
	log.Printf("Task.CreateContext: name=%q", name)
	
	// Debug: log resources map
	log.Printf("Task.CreateContext: resources map has %d entries", len(d.resources))
	for k := range d.resources {
		log.Printf("  - resource: %q", k)
	}
	
	// Handle task resource paths directly (e.g., "1/ctl", "2/cmd")
	if idx := strings.IndexByte(name, '/'); idx > 0 {
		taskID := name[:idx]
		subPath := name[idx+1:]
		
		// Check if this is a task resource
		if resource, ok := d.resources[taskID]; ok {
			log.Printf("Task.CreateContext: found task %q, creating %q", taskID, subPath)
			if cfs, ok := resource.(fs.CreateFS); ok {
				return cfs.Create(subPath)
			}
			// Fall back to open if create not supported
			return fs.OpenContext(ctx, resource, subPath)
		}
	}
	
	// Handle other paths through ResolveFS
	fsys, rname, err := d.ResolveFS(ctx, name)
	if err != nil {
		log.Printf("Task.CreateContext: ResolveFS error: %v", err)
		return nil, err
	}
	log.Printf("Task.CreateContext: resolved to fsys=%T, rname=%q", fsys, rname)
	if cfs, ok := fsys.(fs.CreateFS); ok {
		file, err := cfs.Create(rname)
		if err != nil {
			log.Printf("Task.CreateContext: Create failed: %v", err)
		}
		return file, err
	}
	return nil, &fs.PathError{Op: "create", Path: name, Err: fs.ErrPermission}
}
