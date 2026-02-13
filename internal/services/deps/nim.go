package deps

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
)

type NimManager struct {
	BaseManager
}

func NewNimManager(language string) *NimManager {
	return &NimManager{
		BaseManager: BaseManager{
			Language:     language,
			InstallCmd:   []string{"nimble", "install", "-y"},
			UninstallCmd: []string{"nimble", "uninstall", "-y"},
			ListCmd:      []string{"nimble", "list", "-i"},
			Separator:    "@",
		},
	}
}

func (m *NimManager) GetInstalledPackages(language, langVersion string) ([]models.Dependency, error) {
	output, err := m.runMiseCommand(langVersion, m.ListCmd)
	if err != nil {
		logger.Warnf("GetInstalledPackages for %s failed: %v", language, err)
	}

	var packages []models.Dependency
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "[") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			packages = append(packages, models.Dependency{Name: fields[0], Language: language})
		}
	}
	return packages, nil
}
