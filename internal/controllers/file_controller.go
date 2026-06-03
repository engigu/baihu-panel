package controllers

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

var (
	extractZip   = utils.ExtractZip
	extractTar   = utils.ExtractTar
	extractTarGz = utils.ExtractTarGz
)

type FileController struct {
	workDir string
}

func NewFileController(workDir string) *FileController {
	os.MkdirAll(workDir, 0755)
	absPath, err := filepath.Abs(workDir)
	if err != nil {
		absPath = workDir
	}
	return &FileController{workDir: absPath}
}

type FileNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	IsDir    bool        `json:"isDir"`
	Children []*FileNode `json:"children,omitempty"`
}

// checkPath 校验路径是否在工作目录内且安全。
// 它返回完整的绝对路径以及一个表示路径是否安全的布尔值。
func (fc *FileController) checkPath(path string, allowRoot bool) (string, bool) {
	fullPath := filepath.Join(fc.workDir, filepath.Clean(path))
	rel, err := filepath.Rel(fc.workDir, fullPath)
	if err != nil {
		return "", false
	}

	// 基础的目录穿越检查
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}

	// 根目录检查
	if !allowRoot && rel == "." {
		return "", false
	}

	return fullPath, true
}

func (fc *FileController) GetFileTree(c *gin.Context) {
	root := &FileNode{
		Name:     filepath.Base(fc.workDir),
		Path:     "",
		IsDir:    true,
		Children: []*FileNode{},
	}

	err := filepath.WalkDir(fc.workDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == fc.workDir {
			return nil
		}

		// 过滤 __pycache__ 文件夹
		if d.IsDir() && d.Name() == "__pycache__" {
			return filepath.SkipDir
		}

		relPath, _ := filepath.Rel(fc.workDir, path)
		parts := strings.Split(relPath, string(filepath.Separator))

		current := root
		for i, part := range parts {
			found := false
			for _, child := range current.Children {
				if child.Name == part {
					current = child
					found = true
					break
				}
			}
			if !found {
				isLast := i == len(parts)-1
				isDir := !isLast || d.IsDir()
				node := &FileNode{
					Name:  part,
					Path:  strings.Join(parts[:i+1], "/"),
					IsDir: isDir,
				}
				if isDir {
					node.Children = []*FileNode{}
				}
				current.Children = append(current.Children, node)
				current = node
			}
		}
		return nil
	})

	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}

	utils.Success(c, root.Children)
}

func (fc *FileController) GetFileContent(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		utils.BadRequest(c, "path参数必填")
		return
	}

	fullPath, safe := fc.checkPath(filePath, false)
	if !safe {
		utils.Forbidden(c, "访问被拒绝")
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		utils.NotFound(c, "文件不存在")
		return
	}

	utils.Success(c, gin.H{
		"path":    filePath,
		"content": string(content),
	})
}

func (fc *FileController) SaveFileContent(c *gin.Context) {
	var req struct {
		Path    string `json:"path" binding:"required"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	fullPath, safe := fc.checkPath(req.Path, false)
	if !safe {
		utils.Forbidden(c, "访问被拒绝")
		return
	}

	os.MkdirAll(filepath.Dir(fullPath), 0755)

	if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
		utils.ServerError(c, err.Error())
		return
	}

	utils.SuccessMsg(c, "保存成功")
}

func (fc *FileController) CreateFile(c *gin.Context) {
	var req struct {
		Path  string `json:"path" binding:"required"`
		IsDir bool   `json:"isDir"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	fullPath, safe := fc.checkPath(req.Path, false)
	if !safe {
		utils.Forbidden(c, "访问被拒绝")
		return
	}

	if req.IsDir {
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			utils.ServerError(c, err.Error())
			return
		}
	} else {
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err := os.WriteFile(fullPath, []byte(""), 0644); err != nil {
			utils.ServerError(c, err.Error())
			return
		}
	}

	utils.SuccessMsg(c, "创建成功")
}

func (fc *FileController) DeleteFile(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	fullPath, safe := fc.checkPath(req.Path, false)
	if !safe {
		utils.Forbidden(c, "访问被拒绝")
		return
	}

	if err := os.RemoveAll(fullPath); err != nil {
		utils.ServerError(c, err.Error())
		return
	}

	utils.SuccessMsg(c, "删除成功")
}

func (fc *FileController) MoveFile(c *gin.Context) {
	var req struct {
		OldPath string `json:"oldPath" binding:"required"`
		NewPath string `json:"newPath" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	oldFull, oldSafe := fc.checkPath(req.OldPath, false)
	newFull, newSafe := fc.checkPath(req.NewPath, false)

	if !oldSafe || !newSafe {
		utils.Forbidden(c, "访问被拒绝")
		return
	}

	if oldFull == newFull {
		utils.Success(c, nil)
		return
	}

	// 检查目标是否存在
	if _, err := os.Stat(newFull); err == nil {
		utils.BadRequest(c, "目标已存在")
		return
	}

	// 确保目标目录存在
	os.MkdirAll(filepath.Dir(newFull), 0755)

	if err := os.Rename(oldFull, newFull); err != nil {
		utils.ServerError(c, err.Error())
		return
	}

	utils.Success(c, nil)
}

func (fc *FileController) CopyFile(c *gin.Context) {
	var req struct {
		SourcePath string `json:"sourcePath" binding:"required"`
		TargetPath string `json:"targetPath" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	sourceFull, sourceSafe := fc.checkPath(req.SourcePath, false)
	targetFull, targetSafe := fc.checkPath(req.TargetPath, false)

	if !sourceSafe || !targetSafe {
		utils.Forbidden(c, "访问被拒绝")
		return
	}

	if sourceFull == targetFull {
		utils.Success(c, nil)
		return
	}

	// Read content
	content, err := os.ReadFile(sourceFull)
	if err != nil {
		utils.NotFound(c, "源文件不存在或无法读取")
		return
	}

	// 确保目标目录存在
	os.MkdirAll(filepath.Dir(targetFull), 0755)

	// 检查目标是否存在
	if _, err := os.Stat(targetFull); err == nil {
		utils.BadRequest(c, "目标已存在")
		return
	}

	if err := os.WriteFile(targetFull, content, 0644); err != nil {
		utils.ServerError(c, err.Error())
		return
	}

	utils.Success(c, nil)
}

func (fc *FileController) RenameFile(c *gin.Context) {
	var req struct {
		OldPath string `json:"oldPath" binding:"required"`
		NewPath string `json:"newPath" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// 校验：重命名禁止跨目录
	if filepath.Dir(filepath.Clean(req.OldPath)) != filepath.Dir(filepath.Clean(req.NewPath)) {
		utils.BadRequest(c, "禁止跨目录重命名")
		return
	}

	oldFull, oldSafe := fc.checkPath(req.OldPath, false)
	newFull, newSafe := fc.checkPath(req.NewPath, false)

	if !oldSafe || !newSafe {
		utils.Forbidden(c, "访问被拒绝")
		return
	}

	if oldFull == newFull {
		utils.Success(c, nil)
		return
	}

	// 检查目标是否存在
	if _, err := os.Stat(newFull); err == nil {
		utils.BadRequest(c, "文件已存在")
		return
	}

	if err := os.Rename(oldFull, newFull); err != nil {
		utils.ServerError(c, err.Error())
		return
	}

	utils.Success(c, nil)
}

// UploadArchive 处理归档文件的上传和解压
func (fc *FileController) UploadArchive(c *gin.Context) {
	targetDir := c.PostForm("path")

	file, err := c.FormFile("file")
	if err != nil {
		utils.BadRequest(c, "请选择文件")
		return
	}

	// 检查文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".zip" && ext != ".tar" && ext != ".gz" && ext != ".tgz" {
		utils.BadRequest(c, "仅支持 zip、tar、gz、tgz 格式")
		return
	}

	// 确定解压目标目录
	extractDir, safe := fc.checkPath(targetDir, true)
	if !safe {
		utils.Forbidden(c, "访问被拒绝")
		return
	}
	os.MkdirAll(extractDir, 0755)

	// 保存临时文件
	// 安全修复：使用 filepath.Base 提取纯文件名，防止路径穿越攻击
	tempFile := filepath.Join(os.TempDir(), filepath.Base(file.Filename))
	if err := c.SaveUploadedFile(file, tempFile); err != nil {
		utils.ServerError(c, "保存文件失败")
		return
	}
	defer os.Remove(tempFile)

	// 解压文件
	var extractErr error
	switch {
	case ext == ".zip":
		extractErr = extractZip(tempFile, extractDir)
	case ext == ".tar":
		extractErr = extractTar(tempFile, extractDir)
	case ext == ".gz" || ext == ".tgz":
		extractErr = extractTarGz(tempFile, extractDir)
	}

	if extractErr != nil {
		utils.ServerError(c, "解压失败: "+extractErr.Error())
		return
	}

	utils.SuccessMsg(c, "导入成功")
}

// UploadFiles 处理多个文件的上传
func (fc *FileController) UploadFiles(c *gin.Context) {
	targetDir := c.PostForm("path")

	// 确定目标目录
	destDir, safe := fc.checkPath(targetDir, true)
	if !safe {
		utils.Forbidden(c, "访问被拒绝")
		return
	}
	os.MkdirAll(destDir, 0755)

	form, err := c.MultipartForm()
	if err != nil {
		utils.BadRequest(c, "请选择文件")
		return
	}

	files := form.File["files"]
	paths := form.Value["paths"] // 相对路径数组，用于保持文件夹结构

	if len(files) == 0 {
		utils.BadRequest(c, "请选择文件")
		return
	}

	for i, file := range files {
		// 获取相对路径（如果有）
		// 安全修复：清理文件名
		relPath := filepath.Base(file.Filename)
		if i < len(paths) && paths[i] != "" {
			relPath = paths[i]
		}

		// 构建完整路径
		fullPath, safe := fc.checkPath(filepath.Join(targetDir, relPath), false)
		if !safe {
			continue
		}

		// 确保父目录存在
		os.MkdirAll(filepath.Dir(fullPath), 0755)

		// 保存文件
		if err := c.SaveUploadedFile(file, fullPath); err != nil {
			utils.ServerError(c, "保存文件失败: "+err.Error())
			return
		}
	}

	utils.SuccessMsg(c, "上传成功")
}

func (fc *FileController) DownloadFile(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		utils.BadRequest(c, "path参数必填")
		return
	}

	fullPath, safe := fc.checkPath(filePath, false)
	if !safe {
		utils.Forbidden(c, "访问被拒绝")
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		utils.NotFound(c, "文件不存在")
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(fullPath))
	c.Header("Content-Type", "application/octet-stream")
	c.File(fullPath)
}

func (fc *FileController) DownloadZip(c *gin.Context) {
	paths := c.QueryArray("path")
	if len(paths) == 0 || c.ContentType() == "application/json" {
		var req struct {
			Paths []string `json:"paths"`
		}
		if err := c.ShouldBindJSON(&req); err == nil && len(paths) == 0 {
			paths = req.Paths
		}
	}

	if len(paths) == 0 {
		utils.BadRequest(c, "path参数必填")
		return
	}

	validatedAbsPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		fullPath, safe := fc.checkPath(path, false)
		if !safe {
			utils.Forbidden(c, "访问被拒绝")
			return
		}

		if _, err := os.Stat(fullPath); err != nil {
			utils.NotFound(c, "文件不存在")
			return
		}

		validatedAbsPaths = append(validatedAbsPaths, fullPath)
	}

	fileName := "baihu-export-" + time.Now().Format("20060102-150405") + ".zip"
	if len(validatedAbsPaths) == 1 {
		fileName = filepath.Base(validatedAbsPaths[0]) + ".zip"
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/zip")

	if err := utils.CreateZip(c.Writer, validatedAbsPaths); err != nil {
		return
	}
}
