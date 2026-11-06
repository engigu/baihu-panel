package deps

import (
	"encoding/json"
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
)

type NodeManager struct {
	BaseManager
}

type npmListJSON struct {
	Dependencies map[string]struct {
		Version string `json:"version"`
	} `json:"dependencies"`
}

func parseNpmListOutput(output []byte, language string) []models.Dependency {
	var res npmListJSON
	if err := json.Unmarshal(output, &res); err != nil {
		return nil
	}
	var packages []models.Dependency
	for name, item := range res.Dependencies {
		if name != "" {
			packages = append(packages, models.Dependency{
				Name:     name,
				Version:  item.Version,
				Language: language,
			})
		}
	}
	return packages
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

	packages := parseNpmListOutput(output, language)
	if len(packages) > 0 {
		return packages, nil
	}

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

func (m *NodeManager) GetInstalledVersion(name, langVersion string) (string, error) {
	output, err := m.runMiseCommand(langVersion, []string{"npm", "list", "-g", name, "--depth=0", "--json"})
	if err == nil {
		var res npmListJSON
		if jsonErr := json.Unmarshal(output, &res); jsonErr == nil {
			if item, ok := res.Dependencies[name]; ok && item.Version != "" {
				return item.Version, nil
			}
		}
	}
	return "", nil
}
