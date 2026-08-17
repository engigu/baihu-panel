package services

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/utils"
)

type WebUIService struct {
	settingsService *SettingsService
}

func NewWebUIService(settingsService *SettingsService) *WebUIService {
	return &WebUIService{
		settingsService: settingsService,
	}
}

// GetActiveWebUIFS 返回当前激活WebUI的 fs.FS 接口。
// 如果激活的WebUI是 "default" 或者不存在，则返回 nil。
func (s *WebUIService) GetActiveWebUIFS() fs.FS {
	activeWebUI := s.settingsService.Get(constant.SectionSite, constant.KeyActiveWebUI)
	if activeWebUI == "" || activeWebUI == "default" {
		return nil
	}

	webuiDir := filepath.Join(constant.DataDir, "webuis", activeWebUI)
	
	// 检查是否存在 uimanifest.json 以确认这是一个有效的WebUI目录
	if _, err := os.Stat(filepath.Join(webuiDir, "uimanifest.json")); os.IsNotExist(err) {
		return nil
	}

	return os.DirFS(webuiDir)
}

// WebUIManifest 代表 uimanifest.json 中的元数据
type WebUIManifest struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	Author          string `json:"author"`
	Description     string `json:"description"`
	MinPanelVersion string `json:"min_panel_version"`
}

// GetWebUIs 获取所有可用的WebUI列表
func (s *WebUIService) GetWebUIs() ([]WebUIManifest, error) {
	// 默认WebUI总是可用的
	webuis := []WebUIManifest{
		{
			Name:        "default",
			Version:     "builtin",
			Author:      "Baihu",
			Description: "内置默认WebUI",
		},
	}

	records := s.settingsService.GetSection("webui")
	for name, val := range records {
		var manifest WebUIManifest
		if err := json.Unmarshal([]byte(val), &manifest); err == nil {
			manifest.Name = name // 强制名称匹配
			webuis = append(webuis, manifest)
		}
	}

	return webuis, nil
}

// ExtractWebUI 将 zip 或 tar.gz 压缩包解压到WebUI目录
func (s *WebUIService) ExtractWebUI(zipPath string) (string, error) {
	// 1. 创建临时解压目录（放在 DataDir 下避免跨分区移动失败）
	baseWebUIDir := filepath.Join(constant.DataDir, "webuis")
	if err := os.MkdirAll(baseWebUIDir, 0755); err != nil {
		return "", fmt.Errorf("无法创建 WebUI 基础目录: %v", err)
	}
	tmpDir, err := os.MkdirTemp(baseWebUIDir, "tmp-webui-*")
	if err != nil {
		return "", fmt.Errorf("无法创建临时解压目录: %v", err)
	}
	// 确保在出错时清理临时目录
	defer os.RemoveAll(tmpDir)

	// 2. 根据后缀名选择解压方法
	var extractErr error
	if strings.HasSuffix(strings.ToLower(zipPath), ".tar.gz") || strings.HasSuffix(strings.ToLower(zipPath), ".tgz") {
		extractErr = utils.ExtractTarGz(zipPath, tmpDir)
	} else {
		extractErr = utils.ExtractZip(zipPath, tmpDir)
	}
	if extractErr != nil {
		return "", fmt.Errorf("解压WebUI包失败: %v", extractErr)
	}

	// 3. 读取并解析 uimanifest.json
	manifestPath := filepath.Join(tmpDir, "uimanifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("webui package must contain a uimanifest.json file")
		}
		return "", fmt.Errorf("无法读取 uimanifest.json: %v", err)
	}

	var webuiManifest WebUIManifest
	if err := json.Unmarshal(manifestData, &webuiManifest); err != nil {
		return "", fmt.Errorf("invalid uimanifest.json format")
	}

	webuiName := webuiManifest.Name
	if webuiName == "" || webuiName == "default" {
		return "", fmt.Errorf("invalid webui name in uimanifest.json")
	}

	// 确保WebUI名称安全，防止目录穿越
	webuiName = filepath.Base(filepath.Clean(webuiName))
	
	// 4. 确保压缩包中包含 index.html 入口文件
	if _, err := os.Stat(filepath.Join(tmpDir, "index.html")); os.IsNotExist(err) {
		return "", fmt.Errorf("webui package must contain an index.html file")
	}

	// 5. 移动临时目录到最终的目标目录
	targetDir := filepath.Join(constant.DataDir, "webuis", webuiName)
	// 如果目标目录已存在，先删除旧版本
	os.RemoveAll(targetDir)
	if err := os.Rename(tmpDir, targetDir); err != nil {
		return "", fmt.Errorf("覆盖安装WebUI失败: %v", err)
	}

	// 6. 将记录保存到 settings 表中
	manifestJSON, _ := json.Marshal(webuiManifest)
	if err := s.settingsService.Set("webui", webuiName, string(manifestJSON)); err != nil {
		// 回滚
		os.RemoveAll(targetDir)
		return "", fmt.Errorf("保存WebUI记录失败: %v", err)
	}

	return webuiName, nil
}

// DeleteWebUI 删除自定义WebUI
func (s *WebUIService) DeleteWebUI(name string) error {
	if name == "" || name == "default" {
		return fmt.Errorf("cannot delete default webui")
	}

	name = filepath.Base(filepath.Clean(name))
	targetDir := filepath.Join(constant.DataDir, "webuis", name)
	
	activeWebUI := s.settingsService.Get(constant.SectionSite, constant.KeyActiveWebUI)
	if activeWebUI == name {
		return fmt.Errorf("cannot delete currently active webui")
	}

	if err := os.RemoveAll(targetDir); err != nil {
		return err
	}

	// 从 settings 表中移除记录
	return s.settingsService.Delete("webui", name)
}

// SetActiveWebUI 设置当前的活动WebUI
func (s *WebUIService) SetActiveWebUI(name string) error {
	if name != "default" {
		name = filepath.Base(filepath.Clean(name))
		targetDir := filepath.Join(constant.DataDir, "webuis", name)
		if _, err := os.Stat(filepath.Join(targetDir, "uimanifest.json")); os.IsNotExist(err) {
			return fmt.Errorf("webui %s not found", name)
		}
	}

	return s.settingsService.Set(constant.SectionSite, constant.KeyActiveWebUI, name)
}
