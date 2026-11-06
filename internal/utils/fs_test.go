package utils

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSafeJoin(t *testing.T) {
	baseDir := filepath.Join("testdir", "safe_base")

	tests := []struct {
		name     string
		subPaths []string
		wantSafe bool
	}{
		{
			name:     "正常单层子路径",
			subPaths: []string{"file.txt"},
			wantSafe: true,
		},
		{
			name:     "正常多层子路径",
			subPaths: []string{"dir1", "dir2", "file.txt"},
			wantSafe: true,
		},
		{
			name:     "包含内部相对折叠但未越界",
			subPaths: []string{"dir1/../dir2/file.txt"},
			wantSafe: true,
		},
		{
			name:     "向上越界逃逸父级",
			subPaths: []string{"../escape.txt"},
			wantSafe: false,
		},
		{
			name:     "深层向上越界逃逸",
			subPaths: []string{"dir1", "../../escape.txt"},
			wantSafe: false,
		},
		{
			name:     "多层连续逃逸",
			subPaths: []string{"../../../../etc/passwd"},
			wantSafe: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotSafe := SafeJoin(baseDir, tt.subPaths...)
			if gotSafe != tt.wantSafe {
				t.Fatalf("SafeJoin() gotSafe = %v, wantSafe = %v (path: %s)", gotSafe, tt.wantSafe, gotPath)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal.txt", "normal.txt"},
		{"../../evil.exe", "evil.exe"},
		{"..\\..\\evil.sh", "evil.sh"},
		{"/var/log/syslog", "syslog"},
		{"C:\\Windows\\System32\\cmd.exe", "cmd.exe"},
		{"", ""},
		{".", ""},
		{"..", ""},
	}

	for _, tt := range tests {
		result := SanitizeFilename(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeFilename(%q) = %q; want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractZip_SymlinkProtection(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "baihu-symlink-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempBase)

	destDir := filepath.Join(tempBase, "dest")
	os.MkdirAll(destDir, 0755)

	outsideSecret := filepath.Join(tempBase, "secret.txt")
	os.WriteFile(outsideSecret, []byte("sensitive_data"), 0644)

	// 在 destDir 中预先建立一个指向外部文件的软链接 (如果系统支持)
	symlinkPath := filepath.Join(destDir, "target.txt")
	symlinkCreated := false
	if err := os.Symlink(outsideSecret, symlinkPath); err == nil {
		symlinkCreated = true
	}

	// 构造包含 target.txt 文件的 zip 包
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// 构造同名常规文件条目
	f1, err := zw.Create("target.txt")
	if err != nil {
		t.Fatalf("创建 Zip 条目失败: %v", err)
	}
	f1.Write([]byte("overwritten_attempt"))

	// 构造显式标记为符号链接的恶意条目
	h2 := &zip.FileHeader{
		Name: "symlink_entry",
	}
	h2.SetMode(0777 | os.ModeSymlink)
	f2, err := zw.CreateHeader(h2)
	if err != nil {
		t.Fatalf("创建 Zip 符号链接条目失败: %v", err)
	}
	f2.Write([]byte("../secret.txt"))

	if err := zw.Close(); err != nil {
		t.Fatalf("关闭 zip.Writer 失败: %v", err)
	}

	zipPath := filepath.Join(tempBase, "test.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("写入测试 zip 文件失败: %v", err)
	}

	if err := ExtractZip(zipPath, destDir); err != nil {
		t.Fatalf("ExtractZip 执行失败: %v", err)
	}

	// 验证：外部敏感文件绝不能被覆盖
	secretData, err := os.ReadFile(outsideSecret)
	if err != nil {
		t.Fatalf("读取外部敏感文件失败: %v", err)
	}
	if string(secretData) != "sensitive_data" {
		t.Errorf("外部敏感文件被符号链接劫持覆盖: %s", string(secretData))
	}

	// 验证：软链接条目未被非法创建为指向外部的符号链接
	if fi, err := os.Lstat(filepath.Join(destDir, "symlink_entry")); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			t.Errorf("非法符号链接条目被创建为软链接")
		}
	}

	_ = symlinkCreated
}


