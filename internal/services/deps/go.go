package deps

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
)

type GoManager struct {
	BaseManager
}

func NewGoManager(language string) *GoManager {
	return &GoManager{
		BaseManager: BaseManager{
			Language:     language,
			InstallCmd:   []string{"go", "install"},
			UninstallCmd: []string{"go", "clean", "-i"},
			ListCmd:      []string{"go", "list", "..."},
			VerifyCmd:    []string{"go", "version"},
			Separator:    "@",
		},
	}
}

func (m *GoManager) GetInstalledPackages(language, langVersion string) ([]models.Dependency, error) {
	output, err := m.runMiseCommand(langVersion, m.ListCmd)
	if err != nil {
		logger.Warnf("GetInstalledPackages for %s failed: %v", language, err)
	}

	var packages []models.Dependency
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Tool") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			packages = append(packages, models.Dependency{Name: fields[0], Language: language})
		}
	}
	return packages, nil
}
