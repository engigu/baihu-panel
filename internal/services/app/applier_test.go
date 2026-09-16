package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/engigu/baihu-panel/internal/models"
)

func TestRunSetup_FailFastOnIntermediateError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "baihu-applier-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	applier := DefaultApplier
	var out bytes.Buffer

	// 模拟中间某一步失败的多行脚本
	// 如果没有 Fail-Fast，最后一行 echo 会将退出码刷成 0
	manifest := &AppManifest{
		ID:   "test-fail-app",
		Name: "测试失败应用",
		Setup: AppSetup{
			Install: `
echo ">> 步骤 1: 开始安装"
cmd /c "exit 1"
echo ">> 步骤 2: 这一行绝对不应该被执行！"
`,
		},
	}

	_, err = applier.runSetup(manifest, tempDir, false, false, &out, func(format string, args ...interface{}) {
		fmt.Fprintf(&out, format+"\n", args...)
	})
	if err == nil {
		t.Fatalf("期望 runSetup 因中间命令失败而直接终止返回错误，但实际返回 nil！输出:\n%s", out.String())
	}

	outputStr := out.String()
	t.Logf("捕获到预期的安装失败错误: %v\n输出:\n%s", err, outputStr)

	if strings.Contains(outputStr, "这一行绝对不应该被执行") {
		t.Fatalf("错误未直接终止！后续命令仍然被执行了！")
	}
}

func TestRunSetup_ArtifactCheckValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "baihu-applier-test-artifact-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	applier := DefaultApplier
	var out bytes.Buffer

	// 安装脚本成功执行，但未按要求产出 bin 目录
	manifest := &AppManifest{
		ID:   "test-artifact-app",
		Name: "测试产物校验应用",
		Setup: AppSetup{
			Install: `echo ">> 没有生成 bin 目录"`,
		},
		SyncRules: &AppSyncRules{
			Defaults: AppTaskDefaults{
				WorkDir: "{app_dir}/bin",
			},
			Tasks: []AppTaskItem{
				{ID: "run", Name: "运行任务"},
			},
		},
	}

	_, err = applier.runSetup(manifest, tempDir, false, false, &out, func(format string, args ...interface{}) {})
	if err == nil {
		t.Fatalf("期望因缺少预期预编译产物目录而校验失败终止，但实际成功返回！")
	}
	t.Logf("成功触发预期产物断言拦截: %v", err)

	// 现在创建 bin 目录并在其内生成产物文件，再次测试应该通过
	binDir := filepath.Join(tempDir, "bin")
	_ = os.MkdirAll(binDir, 0755)
	_ = os.WriteFile(filepath.Join(binDir, "app.dll"), []byte("mock binary"), 0644)

	out.Reset()
	_, err = applier.runSetup(manifest, tempDir, false, false, &out, func(format string, args ...interface{}) {})
	if err != nil {
		t.Fatalf("在产物目录与文件齐全的情况下应该通过，但报错: %v", err)
	}
}

func TestRunSetup_ForceSetup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "baihu-applier-test-force-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	applier := DefaultApplier
	var out bytes.Buffer

	// 配置一个探活总是成功的 check，但开启 forceSetup 时必须强制执行 install
	manifest := &AppManifest{
		ID:   "test-force-app",
		Name: "测试强制重构",
		Setup: AppSetup{
			Check:   `echo "check success"`,
			Install: `echo ">> install executed successfully"`,
		},
	}

	testLog := func(format string, args ...interface{}) {
		fmt.Fprintf(&out, format+"\n", args...)
	}

	// 1. 正常模式：check 成功跳过 install
	skipped, err := applier.runSetup(manifest, tempDir, false, false, &out, testLog)
	if err != nil || !skipped {
		t.Fatalf("正常模式下探活成功应跳过安装，但实际 skipped=%v, err=%v", skipped, err)
	}
	if strings.Contains(out.String(), "install executed successfully") {
		t.Fatalf("正常模式探活成功时不应执行 install")
	}

	// 2. 强制重构模式 (forceSetup=true)：必须跳过探活直接执行 install
	out.Reset()
	skipped, err = applier.runSetup(manifest, tempDir, false, true, &out, testLog)
	if err != nil || skipped {
		t.Fatalf("强制模式下应执行安装，但实际 skipped=%v, err=%v", skipped, err)
	}
	if !strings.Contains(out.String(), "install executed successfully") {
		t.Fatalf("强制模式下必须执行 install，但未找到输出:\n%s", out.String())
	}
}

func TestOrchestrateTask_MultiLanguagesParsing(t *testing.T) {
	langStr := "  dotnet@8.0.425   node@22.0.0 python@3.12   golang  "
	var taskLangs models.TaskLanguages
	for _, item := range strings.Fields(langStr) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "@", 2)
		langName := parts[0]
		langVer := ""
		if len(parts) > 1 {
			langVer = parts[1]
		}
		taskLangs = append(taskLangs, map[string]string{
			"name":    langName,
			"version": langVer,
		})
	}

	if len(taskLangs) != 4 {
		t.Fatalf("预期解析 4 个语言配置，实际得到: %d", len(taskLangs))
	}
	if taskLangs[0]["name"] != "dotnet" || taskLangs[0]["version"] != "8.0.425" {
		t.Errorf("dotnet 解析错误: %+v", taskLangs[0])
	}
	if taskLangs[1]["name"] != "node" || taskLangs[1]["version"] != "22.0.0" {
		t.Errorf("node 解析错误: %+v", taskLangs[1])
	}
	if taskLangs[2]["name"] != "python" || taskLangs[2]["version"] != "3.12" {
		t.Errorf("python 解析错误: %+v", taskLangs[2])
	}
	if taskLangs[3]["name"] != "golang" || taskLangs[3]["version"] != "" {
		t.Errorf("golang 解析错误: %+v", taskLangs[3])
	}
}

