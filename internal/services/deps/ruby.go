package deps

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
)

type RubyManager struct {
	BaseManager
}

func NewRubyManager(language string) *RubyManager {
	return &RubyManager{
		BaseManager: BaseManager{
			Language:     language,
			InstallCmd:   []string{"gem", "install"},
			UninstallCmd: []string{"gem", "uninstall", "-a", "-x"},
			ListCmd:      []string{"gem", "list", "--local"},
			Separator:    " ",
		},
	}
}

func (m *RubyManager) GetInstalledPackages(language, langVersion string) ([]models.Dependency, error) {
	output, err := m.runMiseCommand(langVersion, m.ListCmd)
	if err != nil {
		logger.Warnf("GetInstalledPackages for %s failed: %v", language, err)
	}

	var packages []models.Dependency
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Tool") || strings.HasPrefix(line, "(") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			packages = append(packages, models.Dependency{Name: fields[0], Language: language})
		}
	}
	return packages, nil
}
