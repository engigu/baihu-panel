package utils

import (
	"io"
	"os"
	"path/filepath"

	yeka_zip "github.com/yeka/zip"
)

type ZipWriter struct {
	zw       *yeka_zip.Writer
	password string
}

func NewZipWriter(w io.Writer, password string) *ZipWriter {
	return &ZipWriter{
		zw:       yeka_zip.NewWriter(w),
		password: password,
	}
}

func (zw *ZipWriter) Create(name string) (io.Writer, error) {
	if zw.password != "" {
		return zw.zw.Encrypt(name, zw.password, yeka_zip.AES256Encryption)
	}
	return zw.zw.Create(name)
}

func (zw *ZipWriter) Close() error {
	return zw.zw.Close()
}

func (zw *ZipWriter) AddDir(srcDir, prefix string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		zipPath := filepath.ToSlash(filepath.Join(prefix, relPath))
		if info.IsDir() {
			// Directories are typically implicit or created as 0 byte entries.
			// yeka_zip.Create does not support creating directories explicitly well enough
			// but we can just skip it, because most extractors create them when they see a file.
			return nil
		}
		w, err := zw.Create(zipPath)
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(w, file)
		return err
	})
}

// ZipOpenReader is a wrapper for reading encrypted zips
type ZipReadCloser struct {
	rc       *yeka_zip.ReadCloser
	password string
}

func OpenZipReader(name string, password string) (*ZipReadCloser, error) {
	rc, err := yeka_zip.OpenReader(name)
	if err != nil {
		return nil, err
	}
	return &ZipReadCloser{
		rc:       rc,
		password: password,
	}, nil
}

func (zrc *ZipReadCloser) Close() error {
	return zrc.rc.Close()
}

func (zrc *ZipReadCloser) GetFiles() []*yeka_zip.File {
	return zrc.rc.File
}

func (zrc *ZipReadCloser) OpenFile(f *yeka_zip.File) (io.ReadCloser, error) {
	if f.IsEncrypted() {
		f.SetPassword(zrc.password)
	}
	return f.Open()
}

// ExtractDir 提取目录内容到指定位置
func (zrc *ZipReadCloser) ExtractDir(prefix string, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	for _, f := range zrc.rc.File {
		if len(f.Name) > len(prefix) && f.Name[:len(prefix)] == prefix {
			relPath := f.Name[len(prefix):]
			if relPath == "" {
				continue
			}
			fpath := filepath.Join(destDir, relPath)
			if f.FileInfo().IsDir() {
				if err := os.MkdirAll(fpath, 0755); err != nil {
					return err
				}
				continue
			}
			if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(fpath)
			if err != nil {
				return err
			}
			rc, err := zrc.OpenFile(f)
			if err != nil {
				outFile.Close()
				return err
			}
			_, err = io.Copy(outFile, rc)
			rc.Close()
			outFile.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
