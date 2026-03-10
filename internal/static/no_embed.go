//go:build !web

package static

import (
	"io/fs"
	"os"
)

func GetFS() fs.FS {
	return nil
}

func ReadFile(name string) ([]byte, error) {
	return nil, os.ErrNotExist
}
