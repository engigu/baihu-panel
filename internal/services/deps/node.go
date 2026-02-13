package deps

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
)

type NodeManager struct {
	BaseManager
}

func NewNodeManager(language string) *NodeManager {
	return &NodeManager{
		BaseManager: BaseManager{
			Language:     language,
			InstallCmd:   []string{"npm", "install", "-g"},
			UninstallCmd: []string{"npm", "uninstall", "-g"},
			ListCmd:      []string{"npm", "list", "-g", "--depth=0", "--json"},
			VerifyCmd:    []string{"node", "-v"},
			Separator:    "@",
		},
	}
}

func (m *NodeManager) GetInstalledPackages(language, langVersion string) ([]models.Dependency, error) {
	output, err := m.runMiseCommand(langVersion, m.ListCmd)
	if err != nil {
		logger.Warnf("GetInstalledPackages for %s failed: %v", language, err)
	}

	var packages []models.Dependency
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, `"version"`) || strings.HasPrefix(line, "{") || strings.HasPrefix(line, "}") || strings.HasPrefix(line, "]") {
			continue
		}
		if strings.HasPrefix(line, `"`) && strings.Contains(line, ":") {
			name := strings.Trim(strings.Split(line, ":")[0], `" `)
			if name != "" && name != "dependencies" && name != "name" {
				packages = append(packages, models.Dependency{Name: name, Language: language})
			}
		}
	}
	return packages, nil
}
