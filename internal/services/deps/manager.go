package deps

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
)

// Manager 依赖管理器接口
type Manager interface {
	Install(dep *models.Dependency) error
	Uninstall(dep *models.Dependency) error
	GetInstalledPackages(language, langVersion string) ([]models.Dependency, error)
	GetInstallCommand(dep *models.Dependency) (string, error)
	GetReinstallAllCommand(deps []models.Dependency) (string, error)
}

// BaseManager 基础管理器，提供通用方法
type BaseManager struct {
	Language     string
	InstallCmd   []string
	UninstallCmd []string
	ListCmd      []string
	Separator    string
}

func (m *BaseManager) runMiseCommand(langVersion string, cmdArgs []string) ([]byte, error) {
	var args []string
	if langVersion != "" {
		args = []string{"mise", "exec", m.Language + "@" + langVersion, "--"}
	}
	args = append(args, cmdArgs...)

	cmd := exec.Command(args[0], args[1:]...)
	return cmd.CombinedOutput()
}

func (m *BaseManager) Install(dep *models.Dependency) error {
	var packageSpec string
	if dep.Version != "" {
		packageSpec = dep.Name + m.Separator + dep.Version
	} else {
		packageSpec = dep.Name
	}

	args := append([]string{}, m.InstallCmd...)
	args = append(args, packageSpec)

	logger.Infof("Installing %s package: %s", m.Language, packageSpec)
	output, err := m.runMiseCommand(dep.LangVersion, args)
	dep.Log = string(output)

	if err != nil {
		logger.Errorf("Install failed: %v, output: %s", err, string(output))
		return errors.New("安装失败: " + string(output))
	}
	logger.Infof("Install success: %s", packageSpec)
	return nil
}

func (m *BaseManager) GetInstallCommand(dep *models.Dependency) (string, error) {
	var packageSpec string
	if dep.Version != "" {
		packageSpec = dep.Name + m.Separator + dep.Version
	} else {
		packageSpec = dep.Name
	}

	var args []string
	if dep.LangVersion != "" {
		args = []string{"mise", "exec", m.Language + "@" + dep.LangVersion, "--"}
	}
	args = append(args, m.InstallCmd...)
	args = append(args, packageSpec)

	fullCmd := strings.Join(args, " ")
	return fullCmd + " && echo \"__INSTALL_SUCCESS__\" || echo \"__INSTALL_FAILED__\"", nil
}

func (m *BaseManager) GetReinstallAllCommand(deps []models.Dependency) (string, error) {
	if len(deps) == 0 {
		return "echo \"没有需要安装的依赖\"", nil
	}

	var packageSpecs []string
	var langVersion string
	for _, dep := range deps {
		if dep.LangVersion != "" {
			langVersion = dep.LangVersion
		}
		if dep.Version != "" {
			packageSpecs = append(packageSpecs, dep.Name+m.Separator+dep.Version)
		} else {
			packageSpecs = append(packageSpecs, dep.Name)
		}
	}

	var args []string
	if langVersion != "" {
		args = []string{"mise", "exec", m.Language + "@" + langVersion, "--"}
	}
	args = append(args, m.InstallCmd...)
	args = append(args, packageSpecs...)

	fullCmd := strings.Join(args, " ")
	return fullCmd + " && echo \"__INSTALL_SUCCESS__\" || echo \"__INSTALL_FAILED__\"", nil
}

func (m *BaseManager) Uninstall(dep *models.Dependency) error {
	args := append([]string{}, m.UninstallCmd...)
	args = append(args, dep.Name)

	logger.Infof("Uninstalling %s package: %s", m.Language, dep.Name)
	output, err := m.runMiseCommand(dep.LangVersion, args)
	if err != nil {
		logger.Errorf("Uninstall failed: %v, output: %s", err, string(output))
		return errors.New("卸载失败: " + string(output))
	}
	return nil
}

// GetManager 根据语言获取对应的管理器
func GetManager(language string) Manager {
	lang := strings.ToLower(language)
	if strings.Contains(lang, "python") {
		return NewPythonManager(language)
	}
	if strings.Contains(lang, "node") {
		return NewNodeManager(language)
	}
	if strings.Contains(lang, "ruby") {
		return NewRubyManager(language)
	}
	if strings.Contains(lang, "go") {
		return NewGoManager(language)
	}
	if strings.Contains(lang, "rust") {
		return NewRustManager(language)
	}
	if strings.Contains(lang, "bun") {
		return NewBunManager(language)
	}
	if strings.Contains(lang, "php") {
		return NewPhpManager(language)
	}
	if strings.Contains(lang, "deno") {
		return NewDenoManager(language)
	}
	if strings.Contains(lang, "dotnet") {
		return NewDotnetManager(language)
	}
	if strings.Contains(lang, "elixir") || strings.Contains(lang, "erlang") {
		return NewElixirManager(language)
	}
	if strings.Contains(lang, "lua") {
		return NewLuaManager(language)
	}
	if strings.Contains(lang, "nim") {
		return NewNimManager(language)
	}
	if strings.Contains(lang, "dart") || strings.Contains(lang, "flutter") {
		return NewDartManager(language)
	}
	if strings.Contains(lang, "perl") {
		return NewPerlManager(language)
	}
	if strings.Contains(lang, "crystal") {
		return NewCrystalManager(language)
	}
	return nil
}
