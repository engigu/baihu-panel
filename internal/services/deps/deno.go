package deps

import (
	"github.com/engigu/baihu-panel/internal/models"
)

type DenoManager struct {
	BaseManager
}

func NewDenoManager(language string) *DenoManager {
	return &DenoManager{
		BaseManager: BaseManager{
			Language:     language,
			InstallCmd:   []string{"deno", "install", "--global", "-A"},
			UninstallCmd: []string{"deno", "uninstall"},
			ListCmd:      []string{"ls", "-1", "/root/.deno/bin"}, // 这是一个简单的 fallback 尝试
			Separator:    "@",
		},
	}
}

func (m *DenoManager) GetInstalledPackages(language, langVersion string) ([]models.Dependency, error) {
	// Deno 没有官方的 list 命令，暂时留空或通过文件系统猜测
	return []models.Dependency{}, nil
}
