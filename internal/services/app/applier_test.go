package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/relation"
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

func TestRemoveApp_CleanEnvsAndOrphanTags(t *testing.T) {
	tempDBFile, err := os.CreateTemp("", "baihu-app-uninstall-test-*.db")
	if err != nil {
		t.Fatalf("创建临时数据库失败: %v", err)
	}
	tempDBPath := tempDBFile.Name()
	tempDBFile.Close()
	defer os.RemoveAll(tempDBPath)

	if err := database.Init(&database.Config{
		Type: "sqlite",
		Path: tempDBPath,
	}); err != nil {
		t.Fatalf("初始化 sqlite 失败: %v", err)
	}
	if err := database.AutoMigrate(
		&models.Task{},
		&models.EnvironmentVariable{},
		&models.DataStorage{},
		&models.DataRelation{},
		&models.NotifyBinding{},
	); err != nil {
		t.Fatalf("迁移表结构失败: %v", err)
	}

	rawYAML := `spec_version: "v1"
id: "demo-clean-app"
name: "卸载清理测试应用"
version: "1.0.0"
author: "tester"
category: "系统工具"
template:
  - tag: "DemoCleanTag"
  - mise_languages: "node@23"
setup:
  install: "echo ok"
env_schema:
  - key: "DEMO_EXCLUSIVE_TOKEN"
    label: "专属Token"
    type: "string"
    default: "123456"
  - key: "DEMO_SHARED_COOKIE"
    label: "共享Cookie"
    type: "string"
    default: "shared_val"
tasks:
  - id: "sub1"
    name: "子任务1"
    command: "node index.js"
    default_cron: "0 0 8 * * *"
    enabled: true
`
	manifest, err := ParseManifestFromYAML([]byte(rawYAML))
	if err != nil {
		t.Fatalf("解析 manifest 失败: %v", err)
	}

	var out bytes.Buffer
	res, err := DefaultApplier.Apply(manifest, []byte(rawYAML), ApplyOptions{
		SkipSetup: true,
		SkipSync:  true,
		LogWriter: &out,
	})
	if err != nil {
		t.Fatalf("部署应用失败: %v", err)
	}

	// 验证安装后存在主任务、子任务、2个环境变量、以及 task_tag 和 env_tag (DemoCleanTag)
	var taskTagCount, envTagCount, envCount int64
	database.DB.Model(&models.DataStorage{}).Where("type = ? AND name = ?", constant.RelationTypeTaskTag, "DemoCleanTag").Count(&taskTagCount)
	database.DB.Model(&models.DataStorage{}).Where("type = ? AND name = ?", constant.RelationTypeEnvTag, "DemoCleanTag").Count(&envTagCount)
	database.DB.Model(&models.EnvironmentVariable{}).Count(&envCount)
	if taskTagCount != 1 || envTagCount != 1 || envCount != 2 {
		t.Fatalf("安装后数据断言失败: taskTag=%d, envTag=%d, envCount=%d", taskTagCount, envTagCount, envCount)
	}

	// 模拟另一个独立任务绑定了 DEMO_SHARED_COOKIE
	var sharedEnv models.EnvironmentVariable
	database.DB.Where("name = ?", "DEMO_SHARED_COOKIE").First(&sharedEnv)
	relation.DataRelation.SaveRelations("other-standalone-task-id", constant.RelationTypeTaskEnv, sharedEnv.ID)

	// 执行卸载（开启 CleanData 和 CleanEnvs）
	deletedIDs, err := DefaultAppService.RemoveAppWithOptions(res.ID, AppRemoveOptions{
		CleanData: true,
		CleanEnvs: true,
	}, &out)
	if err != nil {
		t.Fatalf("卸载应用失败: %v", err)
	}
	if len(deletedIDs) != 2 {
		t.Fatalf("预期返回 2 个被删除任务 ID (1 主 + 1 子)，实际得到: %v", deletedIDs)
	}

	// 1. 验证任务与子任务全部清空
	var remainingTasks int64
	database.DB.Model(&models.Task{}).Count(&remainingTasks)
	if remainingTasks != 0 {
		t.Errorf("预期任务全部清空，剩余: %d", remainingTasks)
	}

	// 2. 验证 task_tag (DemoCleanTag) 已被自动回收
	database.DB.Model(&models.DataStorage{}).Where("type = ? AND name = ?", constant.RelationTypeTaskTag, "DemoCleanTag").Count(&taskTagCount)
	if taskTagCount != 0 {
		t.Errorf("预期孤儿 task_tag 'DemoCleanTag' 被自动回收，实际剩余: %d", taskTagCount)
	}

	// 3. 验证专属环境变量 DEMO_EXCLUSIVE_TOKEN 被删除，而共享变量 DEMO_SHARED_COOKIE 被安全保留
	var exclusiveCount, sharedCount int64
	database.DB.Model(&models.EnvironmentVariable{}).Where("name = ?", "DEMO_EXCLUSIVE_TOKEN").Count(&exclusiveCount)
	database.DB.Model(&models.EnvironmentVariable{}).Where("name = ?", "DEMO_SHARED_COOKIE").Count(&sharedCount)
	if exclusiveCount != 0 {
		t.Errorf("预期专属环境变量 DEMO_EXCLUSIVE_TOKEN 被删除，实际剩余: %d", exclusiveCount)
	}
	if sharedCount != 1 {
		t.Errorf("预期被其他任务引用的共享环境变量 DEMO_SHARED_COOKIE 被保留，实际剩余: %d", sharedCount)
	}

	// 4. 如果再把 DEMO_SHARED_COOKIE 删除，验证 env_tag (DemoCleanTag) 也会被自动回收
	relation.DataRelation.CleanRelations(sharedEnv.ID, constant.RelationTypeEnvTag)
	database.DB.Model(&models.DataStorage{}).Where("type = ? AND name = ?", constant.RelationTypeEnvTag, "DemoCleanTag").Count(&envTagCount)
	if envTagCount != 0 {
		t.Errorf("预期当所有关联环境变量移除后 env_tag 'DemoCleanTag' 自动回收，实际剩余: %d", envTagCount)
	}
}


