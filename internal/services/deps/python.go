package deps

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
)

type PythonManager struct {
	BaseManager
}

func NewPythonManager(language string) *PythonManager {
	return &PythonManager{
		BaseManager: BaseManager{
			Language:     language,
			InstallCmd:   []string{"pip", "install"},
			UninstallCmd: []string{"pip", "uninstall", "-y"},
			ListCmd:      []string{"pip", "list", "--format=freeze"},
			Separator:    "==",
		},
	}
}

func (m *PythonManager) GetInstalledPackages(language, langVersion string) ([]models.Dependency, error) {
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
		parts := strings.SplitN(line, "==", 2)
		pkg := models.Dependency{Name: parts[0], Language: language}
		if len(parts) > 1 {
			pkg.Version = parts[1]
		}
		packages = append(packages, pkg)
	}
	return packages, nil
}
