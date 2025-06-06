package fs

import (
	"context"
	"fmt"
)

type ResolveFS interface {
	FS
	ResolveFS(ctx context.Context, name string) (FS, string, error)
}

// ResolveTo resolves the name to an FS extension type if possible. It uses
// ResolveFS if available, otherwise it falls back to SubFS.
func ResolveTo[T FS](fsys FS, ctx context.Context, name string) (T, string, error) {
	var tfsys T

	rfsys, rname, err := Resolve(fsys, ctx, name)
	if err != nil {
		return tfsys, "", err
	}

	// try to resolve again from here
	if res, ok := rfsys.(ResolveFS); ok {
		rrfsys, rrname, err := res.ResolveFS(ctx, rname)
		if err == nil && (!Equal(rrfsys, rfsys) || rrname != rname) {
			rfsys = rrfsys
			rname = rrname
		}
	}

	if v, ok := rfsys.(T); ok {
		tfsys = v
	} else {
		return tfsys, "", fmt.Errorf("resolve: %w on %T", ErrNotSupported, rfsys)
	}

	return tfsys, rname, nil
}

// Resolve resolves to the FS directly containing the name returning that
// resolved FS and the relative name for that FS. It uses ResolveFS if
// available, otherwise it falls back to SubFS. If unable to resolve,
// it returns the original FS and the original name, but it can also
// return a PathError.
func Resolve(fsys FS, ctx context.Context, name string) (rfsys FS, rname string, err error) {
	currentFS := fsys
	currentName := name

	// Loop to handle recursive resolution.
	for i := 0; i < 100; i++ { // Add a loop limit to prevent infinite recursion
		resolver, ok := currentFS.(ResolveFS)
		if !ok {
			// The current filesystem does not implement ResolveFS, so we're at the leaf.
			return currentFS, currentName, nil
		}

		nextFS, nextName, err := resolver.ResolveFS(ctx, currentName)
		if err != nil {
			return nil, "", err
		}

		// If the filesystem and name haven't changed, resolution has stabilized.
		if Equal(nextFS, currentFS) && nextName == currentName {
			return currentFS, currentName, nil
		}

		// Continue resolving with the new filesystem and path.
		currentFS = nextFS
		currentName = nextName
	}
	return nil, "", fmt.Errorf("resolution depth exceeded for path: %s", name)
}
