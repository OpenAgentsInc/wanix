package fskit

import (
	"context"
	"errors"
	"log"
	"strings"

	"tractor.dev/wanix/fs"
)

// read-only union of filesystems
type UnionFS []fs.FS

func (f UnionFS) Open(name string) (fs.File, error) {
	ctx := fs.WithOrigin(context.Background(), f, name, "open")
	return f.OpenContext(ctx, name)
}

func (f UnionFS) OpenContext(ctx context.Context, name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	rfsys, rname, err := f.ResolveFS(ctx, name)
	if err != nil {
		return nil, err
	}
	if rname != name || !fs.Equal(rfsys, f) {
		return fs.OpenContext(ctx, rfsys, rname)
	}

	if name != "." {
		log.Printf("non-root open: %s (=> %T %s)", name, rfsys, rname)
		// if non-root open and not resolved, it does not exist
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	var entries []fs.DirEntry
	for _, fsys := range f {
		e, err := fs.ReadDirContext(ctx, fsys, name)
		if err != nil {
			log.Printf("readdir: %v %T %s\n", err, fsys, name)
			continue
		}
		entries = append(entries, e...)
	}

	return DirFile(Entry(name, 0555), entries...), nil
}

func (f UnionFS) ResolveFS(ctx context.Context, name string) (fs.FS, string, error) {
	// Debug logging
	if strings.Contains(name, "data") || strings.Contains(name, "ctl") {
		log.Printf("UnionFS.ResolveFS: name=%q, members=%d", name, len(f))
		for i, fsys := range f {
			log.Printf("  - member[%d]: %T", i, fsys)
		}
	}
	
	if len(f) == 0 {
		return nil, "", &fs.PathError{Op: "resolve", Path: name, Err: fs.ErrNotExist}
	}
	if len(f) == 1 {
		return f[0], name, nil
	}
	if name == "." && fs.IsReadOnly(ctx) {
		return f, name, nil
	}

	var toStat []fs.FS
	for _, fsys := range f {
		if resolver, ok := fsys.(fs.ResolveFS); ok {
			rfsys, rname, err := resolver.ResolveFS(ctx, name)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					// certainly does not have name
					continue
				}
				return rfsys, rname, err
			}
			if !fs.IsReadOnly(ctx) {
				if _, ok := rfsys.(fs.CreateFS); ok {
					return rfsys, rname, nil
				}
			}
			if rname != name || !fs.Equal(rfsys, fsys) {
				// certainly does have name
				return rfsys, rname, nil
			}
		}
		toStat = append(toStat, fsys)
	}

	for _, fsys := range toStat {
		_, err := fs.StatContext(ctx, fsys, name)
		if err != nil {
			continue
		}
		if fs.IsReadOnly(ctx) {
			return fsys, name, nil
		}
		if _, ok := fsys.(fs.CreateFS); ok {
			return fsys, name, nil
		}
	}

	return f, name, nil
}

// Create creates or truncates the named file.
func (f UnionFS) Create(name string) (fs.File, error) {
	ctx := fs.WithOrigin(context.Background(), f, name, "create")
	return f.CreateContext(ctx, name)
}

// CreateContext creates or truncates the named file with context.
func (f UnionFS) CreateContext(ctx context.Context, name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "create", Path: name, Err: fs.ErrInvalid}
	}

	// Try each filesystem in order until one can create
	for i, fsys := range f {
		if cfs, ok := fsys.(fs.CreateFS); ok {
			file, err := cfs.Create(name)
			if err == nil {
				return file, nil
			}
			// Debug logging for task ctl files
			if strings.Contains(name, "ctl") {
				log.Printf("UnionFS.Create[%d]: fsys=%T, name=%q, err=%v", i, fsys, name, err)
			}
			// If it's not a "not exist" error, return it
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}
	}

	// If no filesystem could create, try to open instead
	// This handles the case where a file already exists
	for _, fsys := range f {
		file, err := fs.OpenContext(ctx, fsys, name)
		if err == nil {
			return file, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	return nil, &fs.PathError{Op: "create", Path: name, Err: fs.ErrNotExist}
}
