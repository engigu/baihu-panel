package deps

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
)

type DotnetManager struct {
	BaseManager
}

func NewDotnetManager(language string) *DotnetManager {
	return &DotnetManager{
		BaseManager: BaseManager{
			Language:     language,
			InstallCmd:   []string{"dotnet", "tool", "install", "-g"},
			UninstallCmd: []string{"dotnet", "tool", "uninstall", "-g"},
			ListCmd:      []string{"dotnet", "tool", "list", "-g"},
			Separator:    " ",
		},
	}
}

func (m *DotnetManager) GetInstalledPackages(language, langVersion string) ([]models.Dependency, error) {
	output, err := m.runMiseCommand(langVersion, m.ListCmd)
	if err != nil {
		logger.Warnf("GetInstalledPackages for %s failed: %v", language, err)
	}

	var packages []models.Dependency
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if i < 2 || line == "" { // 跳过表头
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			packages = append(packages, models.Dependency{Name: fields[0], Language: language, Version: fields[1]})
		}
	}
	return packages, nil
}
