package fskit

import (
	"context"
	"log"
	"path"
	"slices"
	"strings"

	"tractor.dev/wanix/fs"
)

func getMapKeys(m MapFS) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

type MapFS map[string]fs.FS

var _ fs.FS = MapFS(nil)
var _ fs.CreateFS = MapFS(nil)

func (fsys MapFS) ResolveFS(ctx context.Context, name string) (fs.FS, string, error) {
	// Debug logging for DOM/VM paths
	if strings.Contains(name, "/data") || strings.Contains(name, "/ctl") {
		log.Printf("MapFS.ResolveFS: name=%q, keys=%v", name, getMapKeys(fsys))
	}
	
	subfs, found := fsys[name]
	if found {
		if rfsys, ok := subfs.(fs.ResolveFS); ok {
			return rfsys.ResolveFS(ctx, ".")
		}
		return subfs, ".", nil
	}

	// check subpaths of map dirs
	var keys []string
	for p := range fsys {
		keys = append(keys, p)
	}
	for _, key := range MatchPaths(keys, name) {
		relativePath := strings.Trim(strings.TrimPrefix(name, key), "/")
		if strings.Contains(name, "/data") || strings.Contains(name, "/ctl") {
			log.Printf("MapFS.ResolveFS: matched key=%q, relativePath=%q, fsys[key]=%T", key, relativePath, fsys[key])
		}
		if rfsys, ok := fsys[key].(fs.ResolveFS); ok {
			return rfsys.ResolveFS(ctx, relativePath)
		} else {
			// otherwise, we just resolve to first match
			return fsys[key], relativePath, nil
		}
	}

	return fsys, name, nil
}

func (fsys MapFS) Stat(name string) (fs.FileInfo, error) {
	// log.Println("bare stat:", name)
	return fsys.StatContext(context.Background(), name)
}

func (fsys MapFS) StatContext(ctx context.Context, name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}

	// we implement Stat to try and avoid using Open for Stat
	// since it involves calling Stat on all sub filesystem roots
	// which could lead to stack overflow when there is a cycle.

	if name == "." {
		return Entry(name, fs.ModeDir|0555), nil
	}

	subfs, found := fsys[name]
	if found {
		return fs.StatContext(ctx, subfs, ".")
	}

	file, err := fs.OpenContext(ctx, fsys, name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return file.Stat()
}

func (fsys MapFS) Open(name string) (fs.File, error) {
	// log.Println("bare open:", name)
	ctx := fs.WithOrigin(context.Background(), fsys, name, "open")
	return fsys.OpenContext(ctx, name)
}

func (fsys MapFS) OpenContext(ctx context.Context, name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	subfs, found := fsys[name]
	n, isNode := subfs.(*Node)
	if found && !isNode {
		namedFS := NamedFS(subfs, path.Base(name))
		return fs.OpenContext(ctx, namedFS, ".")
	}
	if found && isNode {
		subfs = NamedFS(subfs, path.Base(name))
		if !n.IsDir() {
			// Ordinary file
			return fs.OpenContext(ctx, subfs, ".")
		}
		// otherwise its a directory entry...
	}

	for p, subfs := range fsys {
		if strings.HasPrefix(name, p+"/") {
			subPath := strings.TrimPrefix(name, p+"/")
			mountPath := strings.TrimSuffix(name, "/"+subPath)
			namedFS := NamedFS(subfs, path.Base(mountPath))
			return fs.OpenContext(ctx, namedFS, subPath)
		}
	}

	// Directory, possibly synthesized.
	// Note that file can be nil here: the map need not contain explicit parent directories for all its files.
	// But file can also be non-nil, in case the user wants to set metadata for the directory explicitly.
	// Either way, we need to construct the list of children of this directory.
	var list []*Node
	// var elem string
	var need = make(map[string]bool)
	if name == "." {
		// elem = "."
		for fname, subfs := range fsys {
			i := strings.Index(fname, "/")
			if i < 0 {
				if fname != "." {
					fi, err := fs.StatContext(ctx, subfs, ".")
					if err != nil {
						continue
					}
					list = append(list, RawNode(fi, fname))
				}
			} else {
				need[fname[:i]] = true
			}
		}
	} else {
		// elem = name[strings.LastIndex(name, "/")+1:]
		prefix := name + "/"
		for fname, subfs := range fsys {
			fi, err := fs.StatContext(ctx, subfs, ".")
			if err != nil {
				continue
			}
			if strings.HasPrefix(fname, prefix) {
				felem := fname[len(prefix):]
				i := strings.Index(felem, "/")
				if i < 0 {
					list = append(list, RawNode(fi, felem))
				} else {
					need[fname[len(prefix):len(prefix)+i]] = true
				}
			}
		}
		// If the directory name is not in the map,
		// and there are no children of the name in the map,
		// then the directory is treated as not existing.
		if n == nil && list == nil && len(need) == 0 {
			return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
		}
	}
	for _, fi := range list {
		delete(need, fi.name)
	}
	for name := range need {
		list = append(list, RawNode(name, fs.FileMode(fs.ModeDir|0555)))
	}
	slices.SortFunc(list, func(a, b *Node) int {
		return strings.Compare(a.name, b.name)
	})

	if n == nil {
		n = RawNode(name, fs.ModeDir|0555)
	} else {
		n.name = name
	}

	var entries []fs.DirEntry
	for _, nn := range list {
		entries = append(entries, nn)
	}
	return DirFile(n, entries...), nil
}

// Create creates or truncates the named file.
func (fsys MapFS) Create(name string) (fs.File, error) {
	ctx := fs.WithOrigin(context.Background(), fsys, name, "create")
	return fsys.CreateContext(ctx, name)
}

// CreateContext creates or truncates the named file with context.
func (fsys MapFS) CreateContext(ctx context.Context, name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "create", Path: name, Err: fs.ErrInvalid}
	}
	
	// Debug logging
	if strings.Contains(name, "ctl") {
		log.Printf("MapFS.CreateContext: name=%q, keys=%v", name, getMapKeys(fsys))
	}

	// Try exact match first
	subfs, found := fsys[name]
	if found {
		if cfs, ok := subfs.(fs.CreateFS); ok {
			return cfs.Create(".")
		}
		// If the subfs doesn't support Create, try to open it
		return fs.OpenContext(ctx, subfs, ".")
	}

	// Try to find a parent filesystem that contains this path
	for p, subfs := range fsys {
		if strings.HasPrefix(name, p+"/") {
			subPath := strings.TrimPrefix(name, p+"/")
			if cfs, ok := subfs.(fs.CreateFS); ok {
				return cfs.Create(subPath)
			}
			// If create not supported but path matches, try open
			return fs.OpenContext(ctx, subfs, subPath)
		}
	}

	// If we can't find a filesystem that can create this file, fail
	return nil, &fs.PathError{Op: "create", Path: name, Err: fs.ErrNotExist}
}
