package deps

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
)

type CrystalManager struct {
	BaseManager
}

func NewCrystalManager(language string) *CrystalManager {
	return &CrystalManager{
		BaseManager: BaseManager{
			Language:     language,
			InstallCmd:   []string{"shards", "install"}, // 主要是针对当前目录，但 mise exec 下可以安装 tools
			UninstallCmd: []string{"rm", "-rf"},         // 这里的逻辑可能不完全正确
			ListCmd:      []string{"shards", "list"},
			Separator:    " ",
		},
	}
}

func (m *CrystalManager) GetInstalledPackages(language, langVersion string) ([]models.Dependency, error) {
	output, err := m.runMiseCommand(langVersion, m.ListCmd)
	if err != nil {
		logger.Warnf("GetInstalledPackages for %s failed: %v", language, err)
	}

	var packages []models.Dependency
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Shards") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 1 {
			packages = append(packages, models.Dependency{Name: fields[1], Language: language})
		}
	}
	return packages, nil
}
