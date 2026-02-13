package deps

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
)

type PhpManager struct {
	BaseManager
}

func NewPhpManager(language string) *PhpManager {
	return &PhpManager{
		BaseManager: BaseManager{
			Language:     language,
			InstallCmd:   []string{"composer", "global", "require"},
			UninstallCmd: []string{"composer", "global", "remove"},
			ListCmd:      []string{"composer", "global", "show", "--name-only"},
			VerifyCmd:    []string{"php", "-v"},
			Separator:    ":",
		},
	}
}

func (m *PhpManager) GetInstalledPackages(language, langVersion string) ([]models.Dependency, error) {
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
