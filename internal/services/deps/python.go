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

func parsePythonVersionFromShow(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "version:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func (m *PythonManager) GetInstalledVersion(name, langVersion string) (string, error) {
	pkgName := strings.TrimSpace(name)
	if idx := strings.Index(pkgName, "["); idx != -1 {
		pkgName = pkgName[:idx]
	}

	output, err := m.runMiseCommand(langVersion, []string{"pip", "show", pkgName})
	if err == nil {
		if ver := parsePythonVersionFromShow(string(output)); ver != "" {
			return ver, nil
		}
	}

	pkgs, err := m.GetInstalledPackages(m.Language, langVersion)
	if err != nil {
		return "", err
	}
	target := strings.ToLower(strings.ReplaceAll(pkgName, "_", "-"))
	for _, pkg := range pkgs {
		pName := strings.ToLower(strings.ReplaceAll(pkg.Name, "_", "-"))
		if pName == target && pkg.Version != "" {
			return pkg.Version, nil
		}
	}
	return "", nil
}
