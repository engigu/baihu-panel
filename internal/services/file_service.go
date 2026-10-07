package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/engigu/baihu-panel/internal/utils"
)

// FileNode 文件树节点元数据
type FileNode struct {
	Name        string      `json:"name"`
	Path        string      `json:"path"`
	IsDir       bool        `json:"isDir"`
	ModTime     int64       `json:"modTime"`
	HasChildren bool        `json:"hasChildren,omitempty"`
	Children    []*FileNode `json:"children,omitempty"`
}

// FileContentVO 文件内容响应数据
type FileContentVO struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	IsBinary bool   `json:"isBinary"`
}

// FileService 脚本工作目录安全读写与文件系统服务
type FileService struct {
	workDir string
}

func NewFileService(workDir string) *FileService {
	_ = os.MkdirAll(workDir, 0755)
	absPath, err := filepath.Abs(workDir)
	if err != nil {
		absPath = workDir
	}
	return &FileService{workDir: absPath}
}

// CheckPath 校验路径是否在安全工作目录内，返回绝对路径与合法性
func (s *FileService) CheckPath(reqPath string, allowRoot bool) (string, bool) {
	fullPath := filepath.Join(s.workDir, filepath.Clean(reqPath))
	rel, err := filepath.Rel(s.workDir, fullPath)
	if err != nil {
		return "", false
	}

	// 目录穿越检查
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}

	// 根目录检查
	if !allowRoot && rel == "." {
		return "", false
	}

	return fullPath, true
}

// GetFileTree 懒加载获取指定目录下的单层直接子项
func (s *FileService) GetFileTree(subPath string) ([]*FileNode, error) {
	targetPath, safe := s.CheckPath(subPath, true)
	if !safe {
		return nil, fmt.Errorf("访问被拒绝: 目标路径超出安全工作目录")
	}

	fi, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*FileNode{}, nil
		}
		return nil, err
	}

	if !fi.IsDir() {
		return nil, fmt.Errorf("目标路径不是目录")
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return nil, err
	}

	nodes := make([]*FileNode, 0, len(entries))
	for _, entry := range entries {
		var modTime int64
		if info, err := entry.Info(); err == nil {
			modTime = info.ModTime().UnixMilli()
		}

		relPath, err := filepath.Rel(s.workDir, filepath.Join(targetPath, entry.Name()))
		if err != nil {
			relPath = filepath.Join(subPath, entry.Name())
		}
		cleanRel := filepath.ToSlash(filepath.Clean(relPath))
		if cleanRel == "." {
			cleanRel = ""
		}

		isDir := entry.IsDir()
		hasChildren := false
		if isDir {
			if subEntries, err := os.ReadDir(filepath.Join(targetPath, entry.Name())); err == nil && len(subEntries) > 0 {
				hasChildren = true
			}
		}

		nodes = append(nodes, &FileNode{
			Name:        entry.Name(),
			Path:        cleanRel,
			IsDir:       isDir,
			ModTime:     modTime,
			HasChildren: hasChildren,
			Children:    []*FileNode{},
		})
	}

	return nodes, nil
}

// SearchFiles 模糊检索文件
func (s *FileService) SearchFiles(keyword string, limit int, onlyFiles bool) ([]*FileNode, error) {
	if keyword == "" {
		return []*FileNode{}, nil
	}
	if limit <= 0 {
		limit = 100
	}

	lowerKeyword := strings.ToLower(keyword)
	results := make([]*FileNode, 0)

	_ = filepath.WalkDir(s.workDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == s.workDir {
			return nil
		}

		isDir := d.IsDir()
		if onlyFiles && isDir {
			return nil
		}

		name := d.Name()
		if strings.Contains(strings.ToLower(name), lowerKeyword) {
			relPath, err := filepath.Rel(s.workDir, path)
			if err != nil {
				return nil
			}
			cleanRel := filepath.ToSlash(filepath.Clean(relPath))

			var modTime int64
			if info, err := d.Info(); err == nil {
				modTime = info.ModTime().UnixMilli()
			}

			results = append(results, &FileNode{
				Name:    name,
				Path:    cleanRel,
				IsDir:   isDir,
				ModTime: modTime,
			})

			if len(results) >= limit {
				return filepath.SkipAll
			}
		}
		return nil
	})

	return results, nil
}

// GetFileContent 读取文件文本内容
func (s *FileService) GetFileContent(filePath string) (*FileContentVO, error) {
	fullPath, safe := s.CheckPath(filePath, false)
	if !safe {
		return nil, fmt.Errorf("访问被拒绝: 目标路径超出安全工作目录")
	}

	isBin, err := utils.IsBinaryFile(fullPath)
	if err != nil {
		return nil, err
	}

	if isBin {
		return &FileContentVO{
			Path:     filePath,
			Content:  "",
			IsBinary: true,
		}, nil
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("文件不存在或无法读取")
	}

	return &FileContentVO{
		Path:     filePath,
		Content:  string(content),
		IsBinary: false,
	}, nil
}

// SaveFileContent 写入文件文本内容
func (s *FileService) SaveFileContent(filePath, content string) error {
	fullPath, safe := s.CheckPath(filePath, false)
	if !safe {
		return fmt.Errorf("访问被拒绝: 目标路径超出安全工作目录")
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}

// CreateFile 创建新文件或目录
func (s *FileService) CreateFile(filePath string, isDir bool) error {
	fullPath, safe := s.CheckPath(filePath, false)
	if !safe {
		return fmt.Errorf("访问被拒绝: 目标路径超出安全工作目录")
	}

	if isDir {
		return os.MkdirAll(fullPath, 0755)
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(""), 0644)
}

// DeleteFile 递归删除指定文件或目录
func (s *FileService) DeleteFile(filePath string) error {
	fullPath, safe := s.CheckPath(filePath, false)
	if !safe {
		return fmt.Errorf("访问被拒绝: 目标路径超出安全工作目录")
	}

	return os.RemoveAll(fullPath)
}
