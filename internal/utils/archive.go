package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func CreateZip(dst io.Writer, basePaths []string) (err error) {
	w := zip.NewWriter(dst)
	defer func() {
		if closeErr := w.Close(); err == nil {
			err = closeErr
		}
	}()

	for _, basePath := range basePaths {
		info, err := os.Lstat(basePath)
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}

		if info.Mode().IsRegular() {
			if err := addZipFile(w, basePath, filepath.Base(basePath), info); err != nil {
				return err
			}
			continue
		}

		if info.IsDir() {
			baseName := filepath.Base(basePath)
			if err := filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.Type()&fs.ModeSymlink != 0 {
					return nil
				}

				rel, err := filepath.Rel(basePath, path)
				if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
					return nil
				}

				name := baseName
				if rel != "." {
					name = filepath.Join(baseName, rel)
				}
				name = filepath.ToSlash(name)

				info, err := d.Info()
				if err != nil {
					return err
				}

				if d.IsDir() {
					header, err := zip.FileInfoHeader(info)
					if err != nil {
						return err
					}
					header.Name = name + "/"
					_, err = w.CreateHeader(header)
					return err
				}

				if info.Mode().IsRegular() {
					return addZipFile(w, path, name, info)
				}
				return nil
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func addZipFile(w *zip.Writer, path, name string, info os.FileInfo) error {
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(name)
	header.Method = zip.Deflate

	writer, err := w.CreateHeader(header)
	if err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(writer, file)
	return err
}

func ExtractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// 安全检查：防止路径遍历 (ZipSlip)
		rel, err := filepath.Rel(dest, fpath)
		if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func ExtractTar(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	return extractTarReader(tar.NewReader(file), dest)
}

func ExtractTarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	return extractTarReader(tar.NewReader(gzr), dest)
}

func extractTarReader(tr *tar.Reader, dest string) error {
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fpath := filepath.Join(dest, header.Name)

		// 安全检查：防止路径遍历
		rel, err := filepath.Rel(dest, fpath)
		if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(fpath, 0755)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
				return err
			}

			outFile, err := os.Create(fpath)
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

			os.Chmod(fpath, os.FileMode(header.Mode))
		}
	}
	return nil
}
