package deps

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
)

type RustManager struct {
	BaseManager
}

func NewRustManager(language string) *RustManager {
	return &RustManager{
		BaseManager: BaseManager{
			Language:     language,
			InstallCmd:   []string{"cargo", "install"},
			UninstallCmd: []string{"cargo", "uninstall"},
			ListCmd:      []string{"cargo", "install", "--list"},
			Separator:    " v",
		},
	}
}

func (m *RustManager) GetInstalledPackages(language, langVersion string) ([]models.Dependency, error) {
	output, err := m.runMiseCommand(langVersion, m.ListCmd)
	if err != nil {
		logger.Warnf("GetInstalledPackages for %s failed: %v", language, err)
	}

	var packages []models.Dependency
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, " v") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			packages = append(packages, models.Dependency{Name: fields[0], Language: language})
		}
	}
	return packages, nil
}
