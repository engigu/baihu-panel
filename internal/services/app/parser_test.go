package app

import (
	"testing"
)

func TestParseManifest(t *testing.T) {
	yamlContent := `
spec_version: "v1"
id: "bilibili-tool-pro"
name: "B站全自动化助手 (BiliBiliToolPro)"
version: "2.1.0"
author: "RayWangQvQ"
category: "福利签到"
description: "测试描述"

sources:
  - id: "main"
    source_type: "git"
    source_url: "https://github.com/RayWangQvQ/BiliBiliToolPro.git"
    branch: "main"
    blacklist: "qinglong/DefaultTasks|.git"
    target_path: "main"

setup:
  check: "dotnet --version"
  install: "mise install dotnet@8"

env_schema:
  - key: "Ray_BiliBiliCookies__0"
    label: "主账号凭证"
    type: "secret"
    required: true
    tag: "BiliBiliToolPro"

tasks:
  - id: "daily"
    name: "每日签到"
    cron: "0 9 * * *"
    command: "dotnet {app_dir}/bin/Ray.BiliBiliTool.Console.dll"
    enabled: true

scenarios:
  - id: "standard"
    name: "标准模式"
    description: "标准"
    default: true
    task_presets:
      daily:
        enabled: true
`

	manifest, err := ParseManifestFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if manifest.ID != "bilibili-tool-pro" {
		t.Errorf("ID 不符合预期: %s", manifest.ID)
	}

	if len(manifest.Sources) != 1 || manifest.Sources[0].ID != "main" {
		t.Errorf("Sources 解析不符合预期: %+v", manifest.Sources)
	}

	if len(manifest.GetTasks()) != 1 || manifest.GetTasks()[0].ID != "daily" {
		t.Errorf("Tasks 解析不符合预期: %+v", manifest.GetTasks())
	}

	if len(manifest.Scenarios) != 1 || manifest.Scenarios[0].ID != "standard" {
		t.Errorf("Scenarios 解析不符合预期")
	}
}

func TestParseManifestWithTemplate(t *testing.T) {
	yamlContent := `
spec_version: "v1"
id: "bilibili-tool-pro"
name: "B站助手"
version: "2.1.0"
author: "RayWangQvQ"
category: "福利签到"
template:
  - tag: "BiliBiliToolPro"
  - app_prefix: "Ray"
description: "基于 {tag} 的应用"

sources:
  - id: "main"
    source_type: "git"
    source_url: "https://github.com/{tag}.git"

env_schema:
  - key: "COOKIE"
    label: "Cookie"
    type: "secret"
    tag: "{tag}"

tasks:
  - id: "daily"
    name: "每日签到"
    tag: "{tag}"
    command: "dotnet {app_prefix}.Console.dll"
`

	manifest, err := ParseManifestFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("带 template 宏替换的 YAML 解析失败: %v", err)
	}

	if manifest.Description != "基于 BiliBiliToolPro 的应用" {
		t.Errorf("Description 宏替换未生效: %s", manifest.Description)
	}

	if len(manifest.Sources) == 0 || manifest.Sources[0].SourceURL != "https://github.com/BiliBiliToolPro.git" {
		t.Errorf("SourceURL 宏替换未生效: %+v", manifest.Sources)
	}

	if len(manifest.EnvSchema) == 0 || manifest.EnvSchema[0].Tag != "BiliBiliToolPro" {
		t.Errorf("EnvSchema.Tag 宏替换未生效: %+v", manifest.EnvSchema)
	}

	tasks := manifest.GetTasks()
	if len(tasks) == 0 || tasks[0].Command != "dotnet Ray.Console.dll" {
		t.Errorf("Tasks.Command 宏替换未生效: %+v", tasks)
	}
	if len(tasks) == 0 || tasks[0].Tag != "BiliBiliToolPro" {
		t.Errorf("Tasks.Tag 宏替换未生效: %+v", tasks)
	}
}

func TestParseManifestWithMultipleLanguages(t *testing.T) {
	yamlContent := `
spec_version: "v1"
id: "multi-lang-app"
name: "多语言测试应用"
version: "1.0.0"
author: "Baihu"
category: "测试"
template:
  - mise_languages: "dotnet@8.0.425 node@22.0.0 python@3.12"
sync_rules:
  defaults:
    languages: "{mise_languages}"
  tasks:
    - id: "multi_task"
      name: "多环境协同任务"
      language: "{mise_languages}"
      command: "python script.py"
`

	manifest, err := ParseManifestFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("带多语言宏替换的 YAML 解析失败: %v", err)
	}

	if manifest.SyncRules.Defaults.Languages != "dotnet@8.0.425 node@22.0.0 python@3.12" {
		t.Errorf("Defaults.Languages 未正确展开: %s", manifest.SyncRules.Defaults.Languages)
	}

	tasks := manifest.GetTasks()
	if len(tasks) == 0 {
		t.Fatalf("未解析出任务")
	}
	if tasks[0].Language != "dotnet@8.0.425 node@22.0.0 python@3.12" {
		t.Errorf("Task[0].Language 未正确展开: %s", tasks[0].Language)
	}
}

func TestParseBiliBiliToolProManifest(t *testing.T) {
	manifestPath := `F:\workspace\baihu-appstore\apps\BiliBiliToolPro\app.yaml`
	m, err := ParseManifestFromFile(manifestPath)
	if err != nil {
		t.Fatalf("解析 BiliBiliToolPro app.yaml 失败: %v", err)
	}

	// 1. 结构完整性校验
	if err := m.Validate(); err != nil {
		t.Fatalf("Validate 校验失败: %v", err)
	}

	// 2. 元数据断言
	if m.ID != "bilibili-tool-pro" {
		t.Errorf("ID 错误: %s", m.ID)
	}
	if m.Version != "2.1.0" {
		t.Errorf("Version 错误: %s", m.Version)
	}

	// 3. 数据提取方法断言
	// 3.1 任务列表与 defaults 继承
	tasks := m.GetTasks()
	if len(tasks) != 13 {
		t.Errorf("期望 13 个任务，实际: %d", len(tasks))
	}
	daily, found := m.GetTask("daily")
	if !found {
		t.Fatalf("未能提取 daily 任务")
	}
	if daily.Tag != "BiliBiliToolPro" {
		t.Errorf("daily 任务 Tag 宏展开提取错误: %s", daily.Tag)
	}
	if daily.Language != "dotnet@8.0.425" {
		t.Errorf("daily 任务 Language 宏展开提取错误: %s", daily.Language)
	}
	if daily.WorkDir != "{app_dir}/bin" {
		t.Errorf("daily 任务 WorkDir 未正确继承 defaults: %s", daily.WorkDir)
	}

	// 3.2 环境变量契约提取
	envs := m.GetEnvSchema()
	if len(envs) != 9 {
		t.Errorf("期望 9 个环境变量契约，实际: %d", len(envs))
	}
	cookieEnv, found := m.GetEnvItem("Ray_BiliBiliCookies__0")
	if !found {
		t.Fatalf("未能提取 Ray_BiliBiliCookies__0 环境变量")
	}
	if cookieEnv.Tag != "BiliBiliToolPro" {
		t.Errorf("环境变量 Tag 宏展开提取错误: %s", cookieEnv.Tag)
	}

	// 3.3 代码源提取
	sources := m.GetSources()
	if len(sources) != 1 || sources[0].ID != "main" {
		t.Errorf("代码源提取错误: %+v", sources)
	}
	mainSrc, found := m.GetSource("main")
	if !found || mainSrc.SourceType != "git" {
		t.Errorf("GetSource(main) 提取错误: %+v", mainSrc)
	}

	// 3.4 场景预设与计算
	scenarios := m.GetScenarios()
	if len(scenarios) != 3 {
		t.Errorf("期望 3 个场景，实际: %d", len(scenarios))
	}
	defSc := m.GetDefaultScenario()
	if defSc == nil || defSc.ID != "standard" {
		t.Errorf("默认场景期望为 standard，实际: %+v", defSc)
	}

	// 测试场景下的任务状态解析
	enabled, cron := m.ResolveScenarioTaskState("standard", "daily")
	if !enabled || cron != "0 0 9 * * *" {
		t.Errorf("ResolveScenarioTaskState(standard, daily) 期望启用并保持默认 cron，实际: enabled=%v, cron=%s", enabled, cron)
	}
	enabled, _ = m.ResolveScenarioTaskState("minimal", "charge")
	if enabled {
		t.Errorf("ResolveScenarioTaskState(minimal, charge) 期望禁用，实际启用")
	}

	// 3.5 元数据摘要提取
	meta := m.ExtractMetadata()
	if meta["id"] != "bilibili-tool-pro" || meta["tasks_count"] != 13 || meta["env_count"] != 9 {
		t.Errorf("ExtractMetadata 摘要提取不符合预期: %+v", meta)
	}

	// 3.6 针对 BiliBiliToolPro 的全量语言提取方法断言
	if !m.HasLanguage("dotnet") {
		t.Errorf("m.HasLanguage('dotnet') 期望为 true")
	}
	allLangNames := m.GetAllLanguageNames()
	if len(allLangNames) != 1 || allLangNames[0] != "dotnet" {
		t.Errorf("GetAllLanguageNames 提取错误: %+v", allLangNames)
	}
	dotnetTasks := m.GetTasksByLanguage("dotnet")
	if len(dotnetTasks) != 13 {
		t.Errorf("GetTasksByLanguage('dotnet') 期望匹配 13 个任务，实际: %d", len(dotnetTasks))
	}
	taskLangs, ok := m.GetTaskLanguages("daily")
	if !ok || len(taskLangs) != 1 || taskLangs[0]["name"] != "dotnet" || taskLangs[0]["version"] != "8.0.425" {
		t.Errorf("GetTaskLanguages('daily') 提取错误: %+v", taskLangs)
	}
}

func TestLanguageExtractionAndValidation(t *testing.T) {
	// 1. 测试不带版本号的语言配置 (如 "python node")
	plainLangs := "python node"
	if err := ValidateLanguageSpec(plainLangs); err != nil {
		t.Errorf("纯语言名配置验证失败: %v", err)
	}
	parsedPlain := ParseLanguageSpec(plainLangs)
	if len(parsedPlain) != 2 {
		t.Fatalf("纯语言名解析长度期望 2，实际: %d", len(parsedPlain))
	}
	if parsedPlain[0]["name"] != "python" || parsedPlain[0]["version"] != "" {
		t.Errorf("python 解析错误: %+v", parsedPlain[0])
	}
	if parsedPlain[1]["name"] != "node" || parsedPlain[1]["version"] != "" {
		t.Errorf("node 解析错误: %+v", parsedPlain[1])
	}

	// 2. 测试混合带版本号配置 (如 "dotnet@8.0.425 node python@3.11")
	mixedLangs := "dotnet@8.0.425  node   python@3.11"
	if err := ValidateLanguageSpec(mixedLangs); err != nil {
		t.Errorf("混合语言配置验证失败: %v", err)
	}
	parsedMixed := ParseLanguageSpec(mixedLangs)
	if len(parsedMixed) != 3 {
		t.Fatalf("混合语言解析长度期望 3，实际: %d", len(parsedMixed))
	}
	if parsedMixed[0]["name"] != "dotnet" || parsedMixed[0]["version"] != "8.0.425" {
		t.Errorf("dotnet 解析错误: %+v", parsedMixed[0])
	}
	if parsedMixed[1]["name"] != "node" || parsedMixed[1]["version"] != "" {
		t.Errorf("node 解析错误: %+v", parsedMixed[1])
	}
	if parsedMixed[2]["name"] != "python" || parsedMixed[2]["version"] != "3.11" {
		t.Errorf("python 解析错误: %+v", parsedMixed[2])
	}

	// 3. 测试 AppTaskItem 上的值提取与判断方法
	task := AppTaskItem{
		ID:       "worker",
		Name:     "数据分析",
		Language: "python@3.11 node",
	}
	if err := task.ValidateLanguage(); err != nil {
		t.Errorf("task.ValidateLanguage 失败: %v", err)
	}
	if !task.HasLanguage("python") || !task.HasLanguage("node") {
		t.Errorf("task.HasLanguage 失败")
	}
	if task.HasLanguage("golang") {
		t.Errorf("task.HasLanguage('golang') 期望为 false")
	}
	v, found := task.GetLanguageVersion("python")
	if !found || v != "3.11" {
		t.Errorf("GetLanguageVersion('python') 提取错误: %s, %v", v, found)
	}
	v, found = task.GetLanguageVersion("node")
	if !found || v != "" {
		t.Errorf("GetLanguageVersion('node') 提取错误: %s, %v", v, found)
	}
	names := task.GetLanguageNames()
	if len(names) != 2 || names[0] != "python" || names[1] != "node" {
		t.Errorf("GetLanguageNames 提取错误: %+v", names)
	}

	// 4. 测试非法格式拦截
	invalidCases := []string{
		"@3.11",          // 缺少语言名称
		"python python", // 重复语言
		"node@18 node@20",// 重复语言
	}
	for _, tc := range invalidCases {
		if err := ValidateLanguageSpec(tc); err == nil {
			t.Errorf("期望 ValidateLanguageSpec('%s') 报错但通过了", tc)
		}
	}
}



