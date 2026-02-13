package deps

import (
	"github.com/engigu/baihu-panel/internal/models"
)

type PerlManager struct {
	BaseManager
}

func NewPerlManager(language string) *PerlManager {
	return &PerlManager{
		BaseManager: BaseManager{
			Language:     language,
			InstallCmd:   []string{"cpanm"},                // 需要系统预装 cpanm
			UninstallCmd: []string{"cpanm", "--uninstall"}, // 部分 cpanm 版本不支持，这是一个占位
			ListCmd:      []string{"perldoc", "-l"},        // 很难列出所有，暂时简单处理
			VerifyCmd:    []string{"perl", "-v"},
			Separator:    " ",
		},
	}
}

func (m *PerlManager) GetInstalledPackages(language, langVersion string) ([]models.Dependency, error) {
	// Perl 依赖列出比较复杂，暂时返回空
	return []models.Dependency{}, nil
}
