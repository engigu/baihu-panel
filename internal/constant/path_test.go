package constant

import (
	"path/filepath"
	"testing"
)

func TestPathNormalizeAndResolve(t *testing.T) {
	// 测试相对路径归一化
	relPath := "apps/RayWangQvQ-bilibili-tool-pro/bin"
	normalized := NormalizeScriptPath(relPath)
	expectedNormalized := ScriptsDirPlaceholder + "/apps/RayWangQvQ-bilibili-tool-pro/bin"
	if normalized != expectedNormalized {
		t.Errorf("NormalizeScriptPath(%q) = %q; want %q", relPath, normalized, expectedNormalized)
	}

	// 测试带占位符的相对路径还原
	resolved := ResolveScriptPath(normalized)
	expectedResolved := filepath.Clean(filepath.Join(ScriptsWorkDir, "apps/RayWangQvQ-bilibili-tool-pro/bin"))
	if resolved != expectedResolved {
		t.Errorf("ResolveScriptPath(%q) = %q; want %q", normalized, resolved, expectedResolved)
	}

	// 测试不带占位符的相对路径运行时还原兜底
	resolvedFallback := ResolveScriptPath(relPath)
	if resolvedFallback != expectedResolved {
		t.Errorf("ResolveScriptPath fallback(%q) = %q; want %q", relPath, resolvedFallback, expectedResolved)
	}
}
