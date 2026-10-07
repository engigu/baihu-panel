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

// CreateTarGz 将指定的文件或目录列表打包为 tar.gz 流写入 dst
// 可选参数 ignoreRules: 基于 gitignore 语法的规则列表，若为空则使用 DefaultGitIgnoreRules
func CreateTarGz(dst io.Writer, basePaths []string, ignoreRules ...[]string) (err error) {
	var rules []string
	if len(ignoreRules) > 0 && len(ignoreRules[0]) > 0 {
		rules = ignoreRules[0]
	} else {
		rules = DefaultGitIgnoreRules
	}
	matcher := CompileGitIgnore(rules)

	gw := gzip.NewWriter(dst)
	defer func() {
		if closeErr := gw.Close(); err == nil {
			err = closeErr
		}
	}()

	tw := tar.NewWriter(gw)
	defer func() {
		if closeErr := tw.Close(); err == nil {
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
			if matcher != nil && matcher.Match(filepath.Base(basePath), false) {
				continue
			}
			if err := addTarFile(tw, basePath, filepath.Base(basePath), info); err != nil {
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

				// 使用 gitignore 规则匹配过滤
				if rel != "." {
					isDir := d.IsDir()
					if matcher != nil && matcher.Match(rel, isDir) {
						if isDir {
							return filepath.SkipDir
						}
						return nil
					}
				}

				tarName := baseName
				if rel != "." {
					tarName = filepath.Join(baseName, rel)
				}
				tarName = filepath.ToSlash(tarName)

				info, err := d.Info()
				if err != nil {
					return err
				}

				header, err := tar.FileInfoHeader(info, "")
				if err != nil {
					return err
				}
				header.Name = tarName

				if d.IsDir() {
					header.Name += "/"
					return tw.WriteHeader(header)
				}

				if info.Mode().IsRegular() {
					return addTarFile(tw, path, tarName, info)
				}
				return nil
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func addTarFile(tw *tar.Writer, path, name string, info os.FileInfo) error {
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(name)

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(tw, file)
	return err
}

// TarMapping 归档映射条目
type TarMapping struct {
	SourcePath string // 源文件/目录路径（绝对物理路径）
	TargetPath string // 目标归档中的相对路径（例如 "jdpro" 或 "my_scripts/test.js"）
}

// CreateTarGzWithMappings 根据路径映射关系将多个文件或目录打包为 tar.gz 流写入 dst
// 可选参数 ignoreRules: 基于 gitignore 语法的规则列表，若为空则使用 DefaultGitIgnoreRules
func CreateTarGzWithMappings(dst io.Writer, mappings []TarMapping, ignoreRules ...[]string) (err error) {
	var rules []string
	if len(ignoreRules) > 0 && len(ignoreRules[0]) > 0 {
		rules = ignoreRules[0]
	} else {
		rules = DefaultGitIgnoreRules
	}
	matcher := CompileGitIgnore(rules)

	gw := gzip.NewWriter(dst)
	defer func() {
		if closeErr := gw.Close(); err == nil {
			err = closeErr
		}
	}()

	tw := tar.NewWriter(gw)
	defer func() {
		if closeErr := tw.Close(); err == nil {
			err = closeErr
		}
	}()

	for _, m := range mappings {
		srcPath := m.SourcePath
		targetBase := filepath.ToSlash(filepath.Clean(m.TargetPath))
		if targetBase == "." || targetBase == "/" {
			targetBase = ""
		} else {
			targetBase = strings.TrimPrefix(targetBase, "/")
		}

		info, err := os.Lstat(srcPath)
		if err != nil {
			// 如果源文件不存在，跳过或记录
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}

		if info.Mode().IsRegular() {
			if matcher != nil && matcher.Match(filepath.Base(srcPath), false) {
				continue
			}
			tarName := targetBase
			if tarName == "" {
				tarName = filepath.Base(srcPath)
			}
			if err := addTarFile(tw, srcPath, tarName, info); err != nil {
				return err
			}
			continue
		}

		if info.IsDir() {
			if err := filepath.WalkDir(srcPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.Type()&fs.ModeSymlink != 0 {
					return nil
				}

				rel, err := filepath.Rel(srcPath, path)
				if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
					return nil
				}

				// 使用 gitignore 规则匹配过滤
				if rel != "." {
					isDir := d.IsDir()
					if matcher != nil && matcher.Match(rel, isDir) {
						if isDir {
							return filepath.SkipDir
						}
						return nil
					}
				}

				var tarName string
				if rel == "." {
					tarName = targetBase
				} else if targetBase != "" {
					tarName = filepath.ToSlash(filepath.Join(targetBase, rel))
				} else {
					tarName = filepath.ToSlash(rel)
				}

				if tarName == "" {
					return nil
				}

				info, err := d.Info()
				if err != nil {
					return err
				}

				header, err := tar.FileInfoHeader(info, "")
				if err != nil {
					return err
				}
				header.Name = tarName

				if d.IsDir() {
					header.Name = strings.TrimSuffix(header.Name, "/") + "/"
					return tw.WriteHeader(header)
				}

				if info.Mode().IsRegular() {
					return addTarFile(tw, path, tarName, info)
				}
				return nil
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

// ExtractTarGzStream 从 io.Reader 流式解包 tar.gz 到指定目录
func ExtractTarGzStream(r io.Reader, dest string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	return extractTarReader(tar.NewReader(gzr), dest)
}

