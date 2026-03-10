//go:build web

package static

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var distFS embed.FS

// GetFS 返回嵌入的 fs.FS
func GetFS() fs.FS {
	subFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil
	}
	return subFS
}

// ReadFile 读取嵌入的文件
func ReadFile(name string) ([]byte, error) {
	return distFS.ReadFile("dist/" + name)
}
