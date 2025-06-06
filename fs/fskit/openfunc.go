package fskit

import (
	"context"
	"io/fs"
)

type OpenFunc func(ctx context.Context, name string) (fs.File, error)

func (f OpenFunc) Open(name string) (fs.File, error) {
	return f(context.Background(), name)
}

func (f OpenFunc) OpenContext(ctx context.Context, name string) (fs.File, error) {
	return f(ctx, name)
}

// Create creates or truncates the named file.
// For OpenFunc, we delegate to the open function since these are typically synthetic files.
func (f OpenFunc) Create(name string) (fs.File, error) {
	return f(context.Background(), name)
}
