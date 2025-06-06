package vfs

import (
	"context"
	"errors"
	"log"
	"path"
	"slices"
	"strings"

	"tractor.dev/wanix/fs"
	"tractor.dev/wanix/fs/fskit"
)

type BindMode int

const (
	ModeAfter   BindMode = 1
	ModeReplace BindMode = 0
	ModeBefore  BindMode = -1
)

// NS represents a namespace with Plan9-style file and directory bindings.
// Todo: figure out how to make this thread safe. Tricky because ResolveFS
// can call back into the namespace.
type NS struct {
	bindings map[string][]bindTarget
	ctx      context.Context
}

// bindTarget represents a reference to a name in a specific filesystem,
// possibly the root of the filesystem.
type bindTarget struct {
	fs   fs.FS
	path string
	fi   fs.FileInfo
}

// fileInfo returns the latest file info for the binding with the given name
func (ref *bindTarget) fileInfo(ctx context.Context, fname string) (*fskit.Node, error) {
	fi, err := fs.StatContext(ctx, ref.fs, ref.path)
	if err != nil {
		return nil, err
	}
	return fskit.RawNode(fi, fname), nil
}

func New(ctx context.Context) *NS {
	fsys := &NS{
		bindings: make(map[string][]bindTarget),
	}
	fsys.ctx = ctx //fs.WithOrigin(ctx, fsys, "", "new")
	return fsys
}

func (ns *NS) Clone(ctx context.Context) *NS {
	b := make(map[string][]bindTarget)
	for k, v := range ns.bindings {
		b[k] = slices.Clone(v)
	}
	return &NS{
		bindings: b,
		ctx:      ctx,
	}
}

func (ns *NS) Context() context.Context {
	return ns.ctx
}

// getKeys is a helper to sort binding keys for deterministic matching.
func getKeys(m map[string][]bindTarget) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (ns *NS) ResolveFS(ctx context.Context, name string) (fs.FS, string, error) {
	// Handle direct bindings first.
	if refs, ok := ns.bindings[name]; ok {
		if len(refs) == 1 {
			// A single, non-union binding.
			return refs[0].fs, refs[0].path, nil
		}
		// A union binding. The namespace itself must handle it.
		return ns, name, nil
	}

	// Find the longest matching parent binding path.
	for _, bindPath := range fskit.MatchPaths(getKeys(ns.bindings), name) {
		if refs, ok := ns.bindings[bindPath]; ok && len(refs) > 0 {
			ref := refs[0] // We only resolve into the first filesystem of a union.

			// Calculate the new path relative to the bound filesystem's root.
			subPath := strings.Trim(strings.TrimPrefix(name, bindPath), "/")
			newPath := path.Join(ref.path, subPath)

			// Return the next filesystem and the new relative path for further resolution by fs.Resolve.
			return ref.fs, newPath, nil
		}
	}

	// If no binding matched, the path must be resolved from the namespace itself.
	return ns, name, nil
}

func (ns *NS) Unbind(src fs.FS, srcPath, dstPath string) error {
	if !fs.ValidPath(srcPath) {
		return &fs.PathError{Op: "unbind", Path: srcPath, Err: fs.ErrNotExist}
	}
	if !fs.ValidPath(dstPath) {
		return &fs.PathError{Op: "unbind", Path: dstPath, Err: fs.ErrNotExist}
	}

	// Resolve the source path first, just like in Bind
	rfsys, rname, err := fs.Resolve(src, fs.ContextFor(ns), srcPath)
	if err != nil {
		return err
	}

	ns.bindings[dstPath] = slices.DeleteFunc(ns.bindings[dstPath], func(ref bindTarget) bool {
		return fs.Equal(ref.fs, rfsys) && ref.path == rname
	})
	if len(ns.bindings[dstPath]) == 0 {
		delete(ns.bindings, dstPath)
	}

	return nil
}

// Bind adds a file or directory to the namespace. If specified, mode is "after" (default), "before", or "replace",
// which controls the order of the bindings.
// TODO: replace mode arg with BindMode enum
func (ns *NS) Bind(src fs.FS, srcPath, dstPath, mode string) error {
	if !fs.ValidPath(srcPath) {
		return &fs.PathError{Op: "bind", Path: srcPath, Err: fs.ErrNotExist}
	}
	if !fs.ValidPath(dstPath) {
		return &fs.PathError{Op: "bind", Path: dstPath, Err: fs.ErrNotExist}
	}

	// Check srcPath, cache the file info
	rfsys, rname, err := fs.Resolve(src, fs.ContextFor(ns), srcPath)
	if err != nil {
		return err
	}
	file, err := rfsys.Open(rname)
	if err != nil {
		return err
	}
	fi, err := file.Stat()
	if err != nil {
		return err
	}
	file.Close()

	ref := bindTarget{fs: rfsys, path: rname, fi: fi}
	switch mode {
	case "", "after":
		ns.bindings[dstPath] = append([]bindTarget{ref}, ns.bindings[dstPath]...)
	case "before":
		ns.bindings[dstPath] = append(ns.bindings[dstPath], ref)
	case "replace":
		ns.bindings[dstPath] = []bindTarget{ref}
	default:
		return &fs.PathError{Op: "bind", Path: mode, Err: fs.ErrInvalid}
	}
	return nil
}

func (ns *NS) Stat(name string) (fs.FileInfo, error) {
	ctx := fs.WithOrigin(ns.ctx, ns, name, "stat")
	return ns.StatContext(ctx, name)
}

func (ns *NS) StatContext(ctx context.Context, name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}

	ctx = fs.WithOrigin(ctx, ns, name, "stat")

	// we implement Stat to try and avoid using Open for Stat
	// since it involves calling Stat on all sub filesystem roots
	// which could lead to stack overflow when there is a cycle.

	if name == "." {
		return fskit.Entry(name, fs.ModeDir|0755), nil
	}

	// Check direct bindings since they don't get resolved by the resolver.
	// todo: again, if there is a direct binding by this name, it might also
	// exist as a subpath of another binding. so this is not correct.
	if refs, exists := ns.bindings[name]; exists {
		for _, ref := range refs {
			fi, err := ref.fileInfo(ctx, path.Base(name))
			if err != nil {
				continue
			}
			return fi, nil
		}
	}

	tfsys, tname, err := fs.ResolveTo[fs.StatContextFS](ns, ctx, name)
	if err != nil && !errors.Is(err, fs.ErrNotSupported) {
		return nil, err
	}
	if err == nil && !fs.Equal(tfsys, ns) {
		return tfsys.StatContext(ctx, tname)
	}

	rfsys, rname, err := fs.Resolve(ns, ctx, name)
	if err != nil {
		return nil, err
	}

	f, err := fs.OpenContext(ctx, rfsys, rname)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Stat()
}

// Open implements fs.FS interface
func (ns *NS) Open(name string) (fs.File, error) {
	ctx := fs.WithOrigin(ns.ctx, ns, name, "open")
	return ns.OpenContext(ctx, name)
}

// OpenContext ...
func (ns *NS) OpenContext(ctx context.Context, name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	ctx = fs.WithOrigin(ctx, ns, name, "open")

	var dir *fskit.Node
	var dirEntries []fs.DirEntry
	var foundDir bool

	// Check direct bindings
	if refs, exists := ns.bindings[name]; exists {
		for _, ref := range refs {
			if ref.fi.IsDir() {
				// directory binding, add entries
				if dir == nil {
					dir = fskit.RawNode(ref.fi, name)
					foundDir = true
				}
				entries, err := fs.ReadDirContext(ctx, ref.fs, ref.path)
				if err != nil {
					log.Println("readdir error:", err)
					return nil, err
				}
				for _, entry := range entries {
					ei, err := entry.Info()
					if err != nil {
						return nil, err
					}
					dirEntries = append(dirEntries, fskit.RawNode(ei))
				}
			} else {
				// file binding
				if file, err := fs.OpenContext(ctx, ref.fs, ref.path); err == nil {
					return file, nil
				}
				continue
			}

		}
	}

	// Check subpaths of bindings
	var bindPaths []string
	for p := range ns.bindings {
		bindPaths = append(bindPaths, p)
	}
	for _, bindPath := range fskit.MatchPaths(bindPaths, name) {
		for _, ref := range ns.bindings[bindPath] {
			relativePath := path.Join(ref.path, strings.Trim(strings.TrimPrefix(name, bindPath), "/"))
			fi, err := fs.StatContext(ctx, ref.fs, relativePath)
			if err != nil {
				continue
			}
			if fi.IsDir() {
				// directory found in under dir binding
				if dir == nil {
					dir = fskit.RawNode(fi, name)
					foundDir = true
				}
				entries, err := fs.ReadDirContext(ctx, ref.fs, relativePath)
				if err != nil {
					log.Println("readdir error:", err)
					return nil, err
				}
				for _, entry := range entries {
					ei, err := entry.Info()
					if err != nil {
						return nil, err
					}
					dirEntries = append(dirEntries, fskit.RawNode(ei))
				}
			} else {
				// file found in under dir binding
				if file, err := fs.OpenContext(ctx, ref.fs, relativePath); err == nil {
					return file, nil
				}
			}
		}
	}

	// Synthesized parent directories
	var need = make(map[string]bool)
	if name == "." {
		for fname, refs := range ns.bindings {
			i := strings.Index(fname, "/")
			if i < 0 {
				if fname != "." {
					for _, ref := range refs {
						if info, err := ref.fileInfo(ctx, fname); err == nil {
							dirEntries = append(dirEntries, info)
						}
					}
				}
			} else {
				need[fname[:i]] = true
			}
		}
	} else {
		prefix := name + "/"
		for fname, refs := range ns.bindings {
			if strings.HasPrefix(fname, prefix) {
				felem := fname[len(prefix):]
				i := strings.Index(felem, "/")
				if i < 0 {
					for _, ref := range refs {
						if info, err := ref.fileInfo(ctx, fname); err == nil {
							dirEntries = append(dirEntries, info)
						}
					}
				} else {
					need[fname[len(prefix):len(prefix)+i]] = true
				}
			}
		}
		// If the name is not binding,
		// and there are no children of the name and no dir was found,
		// then the directory is treated as not existing.
		if dirEntries == nil && len(need) == 0 && !foundDir {
			return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
		}
	}
	for _, fi := range dirEntries {
		delete(need, fi.Name())
	}
	for name := range need {
		dirEntries = append(dirEntries, fskit.Entry(name, fs.ModeDir|0755))
	}
	slices.SortFunc(dirEntries, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})

	return fskit.DirFile(fskit.Entry(name, fs.ModeDir|0755), dirEntries...), nil
}

// Create creates or truncates the named file.
func (ns *NS) Create(name string) (fs.File, error) {
	ctx := fs.WithOrigin(ns.ctx, ns, name, "create")
	return ns.CreateContext(ctx, name)
}

// CreateContext creates or truncates the named file with context.
func (ns *NS) CreateContext(ctx context.Context, name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "create", Path: name, Err: fs.ErrInvalid}
	}
	
	// Debug logging for task paths
	if strings.HasPrefix(name, "task/") {
		log.Printf("NS.CreateContext: name=%q, bindings count=%d", name, len(ns.bindings))
		for bname := range ns.bindings {
			log.Printf("  - binding: %q", bname)
		}
	}

	// First check if this is a direct binding
	if refs, exists := ns.bindings[name]; exists && len(refs) > 0 {
		ref := refs[0] // Use first binding
		if cfs, ok := ref.fs.(fs.CreateFS); ok {
			return cfs.Create(ref.path)
		}
		// Fall back to open if create not supported
		return fs.OpenContext(ctx, ref.fs, ref.path)
	}

	// Check if any binding is a prefix of the requested path
	for bname, refs := range ns.bindings {
		if strings.HasPrefix(name, bname+"/") && len(refs) > 0 {
			ref := refs[0]
			subPath := path.Join(ref.path, strings.TrimPrefix(name, bname+"/"))
			
			// Debug logging for task paths
			if strings.HasPrefix(name, "task/") {
				log.Printf("NS.CreateContext: matched binding %q, ref.fs=%T, ref.path=%q, subPath=%q", 
					bname, ref.fs, ref.path, subPath)
			}
			
			if cfs, ok := ref.fs.(fs.CreateFS); ok {
				return cfs.Create(subPath)
			}
			// Fall back to open if create not supported
			return fs.OpenContext(ctx, ref.fs, subPath)
		}
	}

	// If no binding matches, we can't create the file
	return nil, &fs.PathError{Op: "create", Path: name, Err: fs.ErrNotExist}
}
