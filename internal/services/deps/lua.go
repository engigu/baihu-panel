package deps

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
)

type LuaManager struct {
	BaseManager
}

func NewLuaManager(language string) *LuaManager {
	return &LuaManager{
		BaseManager: BaseManager{
			Language:     language,
			InstallCmd:   []string{"luarocks", "install"},
			UninstallCmd: []string{"luarocks", "remove"},
			ListCmd:      []string{"luarocks", "list"},
			VerifyCmd:    []string{"lua", "-v"},
			Separator:    " ",
		},
	}
}

func (m *LuaManager) GetInstalledPackages(language, langVersion string) ([]models.Dependency, error) {
	output, err := m.runMiseCommand(langVersion, m.ListCmd)
	if err != nil {
		logger.Warnf("GetInstalledPackages for %s failed: %v", language, err)
	}

	var packages []models.Dependency
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Rock") || strings.HasPrefix(line, "--") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			packages = append(packages, models.Dependency{Name: fields[0], Language: language})
		}
	}
	return packages, nil
}
