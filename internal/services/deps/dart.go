package deps

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
)

type DartManager struct {
	BaseManager
}

func NewDartManager(language string) *DartManager {
	return &DartManager{
		BaseManager: BaseManager{
			Language:     language,
			InstallCmd:   []string{"dart", "pub", "global", "activate"},
			UninstallCmd: []string{"dart", "pub", "global", "deactivate"},
			ListCmd:      []string{"dart", "pub", "global", "list"},
			Separator:    " ",
		},
	}
}

func (m *DartManager) GetInstalledPackages(language, langVersion string) ([]models.Dependency, error) {
	output, err := m.runMiseCommand(langVersion, m.ListCmd)
	if err != nil {
		logger.Warnf("GetInstalledPackages for %s failed: %v", language, err)
	}

	var packages []models.Dependency
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			packages = append(packages, models.Dependency{Name: fields[0], Language: language})
		}
	}
	return packages, nil
}
