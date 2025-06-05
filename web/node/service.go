package node

import (
	"embed"
	"io/fs"

	"tractor.dev/wanix/fs/fskit"
)

//go:embed bootstrap.js
var bootstrapFS embed.FS

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) Open(name string) (fs.File, error) {
	switch name {
	case "bootstrap.js":
		data, err := bootstrapFS.ReadFile("bootstrap.js")
		if err != nil {
			return nil, err
		}
		node := fskit.Entry("bootstrap.js", 0444, data)
		return node.Open(".")
	case "ctl":
		// For now, just return a dummy file
		node := fskit.Entry("ctl", 0644, []byte{})
		return node.Open(".")
	default:
		return nil, fs.ErrNotExist
	}
}