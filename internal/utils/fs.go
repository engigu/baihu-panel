package utils

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// CopyPath copies a file or directory from src to dest
func CopyPath(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(src, dest)
	}
	return CopyFile(src, dest)
}

// CopyFile copies a single file from src to dest
func CopyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err == nil {
		os.Chmod(dest, info.Mode())
	}

	return nil
}

func copyDir(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dest, info.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if err := CopyPath(srcPath, destPath); err != nil {
			return err
		}
	}

	return nil
}


// IsBinaryFile 判断指定路径的文件是否为二进制文件
func IsBinaryFile(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// 读取前 1024 字节进行检测
	buf := make([]byte, 1024)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}

	if n == 0 {
		return false, nil
	}

	// 使用 MIME 探测
	sniffSize := n
	if sniffSize > 512 {
		sniffSize = 512
	}
	mimeType := http.DetectContentType(buf[:sniffSize])

	// 如果是图片，我们允许在前端预览，因此不将其作为崩溃的二进制过滤
	if strings.HasPrefix(mimeType, "image/") {
		return false, nil
	}

	// 常见的文本、配置和前端资源在编辑器中是安全的
	if strings.HasPrefix(mimeType, "text/") ||
		mimeType == "application/json" ||
		mimeType == "application/javascript" ||
		mimeType == "application/xml" {
		return false, nil
	}

	// 其他非 text 类型但属于 octet-stream 等，我们通过 NULL 字节判定
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true, nil
		}
	}
	return false, nil
}

// GetBaihuExecutable 统一获取白虎面板主程序/CLI的可执行文件绝对路径
func GetBaihuExecutable() string {
	if exe, err := os.Executable(); err == nil && exe != "" {
		return exe
	}
	return "baihu"
}

// SafeJoin 安全拼接基础目录与相对子路径，防止路径遍历逃逸 (Zip-Slip / Directory Traversal)
// 返回解析后的完整绝对路径与是否在 baseDir 范围内的安全布尔值 (true: 安全且在 baseDir 范围内; false: 发生逃逸或路径非法)
func SafeJoin(baseDir string, subPaths ...string) (string, bool) {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		absBase = filepath.Clean(baseDir)
	}

	parts := append([]string{absBase}, subPaths...)
	fullPath := filepath.Clean(filepath.Join(parts...))

	rel, err := filepath.Rel(absBase, fullPath)
	if err != nil {
		return "", false
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}

	return fullPath, true
}

// SanitizeFilename 清洗上传的文件名，提取纯文件名，防止包含任何相对或绝对路径穿越字符
func SanitizeFilename(filename string) string {
	clean := filepath.Base(filepath.Clean(filename))
	if clean == "." || clean == ".." || clean == string(filepath.Separator) || clean == "/" || clean == "\\" {
		return ""
	}
	return clean
}


