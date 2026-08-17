package deps

import (
	"regexp"
	"strings"
)

// Detector 依赖检测器接口
type Detector interface {
	Detect(logContent string) []string
}

// PythonDetector Python 依赖检测器
type PythonDetector struct{}

func (d *PythonDetector) Detect(logContent string) []string {
	var pkgs []string
	seen := make(map[string]bool)
	pythonRegex1 := regexp.MustCompile(`ModuleNotFoundError: No module named '([^']+)'`)
	pythonRegex2 := regexp.MustCompile(`No module named ([a-zA-Z0-9_\-]+)`)

	matches := pythonRegex1.FindAllStringSubmatch(logContent, -1)
	for _, m := range matches {
		if len(m) > 1 {
			name := strings.TrimSpace(m[1])
			if name != "" && !seen[name] {
				seen[name] = true
				pkgs = append(pkgs, name)
			}
		}
	}
	matches2 := pythonRegex2.FindAllStringSubmatch(logContent, -1)
	for _, m := range matches2 {
		if len(m) > 1 {
			name := strings.TrimSpace(m[1])
			if name != "" && !seen[name] {
				seen[name] = true
				pkgs = append(pkgs, name)
			}
		}
	}
	return pkgs
}

// NodeDetector Node.js 依赖检测器
type NodeDetector struct{}

func (d *NodeDetector) Detect(logContent string) []string {
	var pkgs []string
	seen := make(map[string]bool)
	nodeRegex1 := regexp.MustCompile(`Error: Cannot find module '([^']+)'`)
	nodeRegex2 := regexp.MustCompile(`Cannot find module '([^']+)'`)

	matches := nodeRegex1.FindAllStringSubmatch(logContent, -1)
	for _, m := range matches {
		if len(m) > 1 {
			name := strings.TrimSpace(m[1])
			if name != "" && !seen[name] {
				seen[name] = true
				pkgs = append(pkgs, name)
			}
		}
	}
	matches2 := nodeRegex2.FindAllStringSubmatch(logContent, -1)
	for _, m := range matches2 {
		if len(m) > 1 {
			name := strings.TrimSpace(m[1])
			if name != "" && !seen[name] {
				seen[name] = true
				pkgs = append(pkgs, name)
			}
		}
	}
	return pkgs
}

var languageDetectors = map[string]Detector{
	"python":  &PythonDetector{},
	"python3": &PythonDetector{},
	"node":    &NodeDetector{},
	"js":      &NodeDetector{},
	"ts":      &NodeDetector{},
	"bun":     &NodeDetector{},
}

// DetectMissingDependencies 从日志内容中检测缺失的依赖包名
func DetectMissingDependencies(language, logContent string) ([]string, bool) {
	lang := strings.ToLower(language)
	var det Detector
	for key, d := range languageDetectors {
		if strings.Contains(lang, key) {
			det = d
			break
		}
	}

	if det == nil {
		return nil, false
	}

	pkgs := det.Detect(logContent)
	return pkgs, len(pkgs) > 0
}
