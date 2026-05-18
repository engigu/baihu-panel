package deps

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/engigu/baihu-panel/internal/models"
)

// ParseManifest 根据语言解析依赖清单文件内容
func ParseManifest(language, content string) ([]models.Dependency, error) {
	lang := strings.ToLower(language)
	if strings.Contains(lang, "python") {
		return ParseRequirements(content), nil
	}
	if strings.Contains(lang, "node") {
		return ParsePackageJson(content)
	}
	return []models.Dependency{}, nil
}

// ParseRequirements 解析 Python requirements.txt
func ParseRequirements(content string) []models.Dependency {
	var deps []models.Dependency
	// 使用正则表达式按行分割，兼容 Windows 和 Linux 的换行符
	lines := regexp.MustCompile(`\r?\n`).Split(content, -1)
	
	// 用于分割包名和版本的正则 (支持 ==, >=, <=, ~=, >, <, @)
	versionRegex := regexp.MustCompile(`[=><~@]+`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 忽略空行、注释行以及参数行 (以 - 开头的行如 -i, -r)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}

		// 分割名称与版本
		parts := versionRegex.Split(line, 2)
		name := strings.TrimSpace(parts[0])
		version := ""
		if len(parts) > 1 {
			// 清除可能存在的后续参数，比如 requests==2.31.0 --hash=sha256:...
			versionPart := strings.TrimSpace(parts[1])
			
			// 如果有逗号分隔的多个范围限制，比如 >=1.20,<2.0，只取第一个范围作为参考版本号
			if idx := strings.Index(versionPart, ","); idx != -1 {
				versionPart = versionPart[:idx]
			}

			versionFields := strings.Fields(versionPart)
			if len(versionFields) > 0 {
				version = strings.TrimSpace(versionFields[0])
				// 清除可能残留的首部版本符号
				version = strings.TrimLeft(version, "=><~@ ")
			}
		}

		if name != "" {
			deps = append(deps, models.Dependency{
				Name:     name,
				Version:  version,
				Language: "python3",
			})
		}
	}
	return deps
}

// PackageJson 代表 package.json 的结构定义
type PackageJson struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// ParsePackageJson 解析 Node.js package.json
func ParsePackageJson(content string) ([]models.Dependency, error) {
	var pkg PackageJson
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return nil, err
	}

	var deps []models.Dependency
	collect := func(m map[string]string, isDev bool) {
		for name, versionRange := range m {
			// 移除 npm 常见版本范围修饰符（如 ^1.2.3 或 ~2.3.0，保留底线版本号）
			version := strings.TrimLeft(versionRange, "^~>=<* ")
			remark := ""
			if isDev {
				remark = "devDependencies"
			}
			deps = append(deps, models.Dependency{
				Name:     name,
				Version:  version,
				Language: "node",
				Remark:   remark,
			})
		}
	}

	collect(pkg.Dependencies, false)
	collect(pkg.DevDependencies, true)
	return deps, nil
}
