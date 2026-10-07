package agentsync

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/engigu/baihu-panel/cmd/clibase"
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
)

type Config struct {
	TaskID      string
	AgentID     string
	Mappings    string
	CleanTarget bool
	IgnoreRules string
	Timeout     int
}

// mappingSlice 支持多次传入 --mapping 参数
type mappingSlice []string

func (m *mappingSlice) String() string {
	return strings.Join(*m, ",")
}

func (m *mappingSlice) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func Run(args []string) {
	fs := flag.NewFlagSet("agentsync", flag.ExitOnError)
	var cfg Config
	var mappingArgs mappingSlice

	fs.StringVar(&cfg.TaskID, "task-id", "", "目标 AgentSync 任务 ID (可直接根据任务配置进行同步)")
	fs.StringVar(&cfg.AgentID, "agent", "", "目标 Agent 节点标识 (ID 或名称)")
	fs.Var(&mappingArgs, "mapping", "路径映射，格式为 源路径:目标路径 (支持重复指定，如 --mapping a:b --mapping c:d)")
	fs.StringVar(&cfg.Mappings, "mappings", "", "路径映射列表 (逗号分隔，如 a:b,c:d)")
	fs.BoolVar(&cfg.CleanTarget, "clean-target", false, "同步前清空 Agent 目标目录 (默认 false)")
	fs.StringVar(&cfg.IgnoreRules, "ignore", "", "排除过滤规则，逗号分隔 (支持标准 gitignore 规则)")
	fs.IntVar(&cfg.Timeout, "timeout", 10, "任务超时时间 (分钟，默认 10)")

	printHelp := func() {
		fmt.Fprintf(os.Stderr, "\n白虎面板 Agent 脚本与目录同步工具 (AgentSync)\n\n")
		fmt.Fprintf(os.Stderr, "用法:\n")
		fmt.Fprintf(os.Stderr, "  baihu agentsync [参数]\n\n")
		fmt.Fprintf(os.Stderr, "参数详情:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  baihu agentsync --task-id dakueb0c5br5cq4rjkv0\n")
		fmt.Fprintf(os.Stderr, "  baihu agentsync --agent node-1 --mapping \"apps/jdpro:jdpro\" --clean-target\n")
		fmt.Fprintf(os.Stderr, "  baihu agentsync --agent node-1 --mapping \"apps/jdpro:jdpro\" --mapping \"common:scripts/common\"\n\n")
	}

	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		printHelp()
		return
	}

	fs.Usage = printHelp
	if err := fs.Parse(args); err != nil {
		return
	}

	// 整合 mappingArgs 和 cfg.Mappings
	var allMappings []string
	for _, m := range mappingArgs {
		trimmed := strings.TrimSpace(m)
		if trimmed != "" {
			allMappings = append(allMappings, trimmed)
		}
	}
	if cfg.Mappings != "" {
		for _, m := range strings.Split(cfg.Mappings, ",") {
			trimmed := strings.TrimSpace(m)
			if trimmed != "" {
				allMappings = append(allMappings, trimmed)
			}
		}
	}

	// 校验参数：必须提供 --task-id 或 (--agent 和至少一个 --mapping)
	if cfg.TaskID == "" && (cfg.AgentID == "" || len(allMappings) == 0) {
		fmt.Fprintf(os.Stderr, "错误: 请指定已有的 --task-id，或者同时提供 --agent 与 --mapping 参数。\n\n")
		fs.Usage()
		return
	}

	// 1. 优先处理 --task-id 模式
	if cfg.TaskID != "" {
		executeByTaskID(cfg.TaskID)
		return
	}

	// 2. 处理直接指定 agent 和 mapping 模式
	executeDirectSync(cfg.AgentID, allMappings, cfg.CleanTarget, cfg.IgnoreRules, cfg.Timeout)
}

func executeByTaskID(taskIDInput string) {
	clibase.InitContext(false)
	taskID := resolveAgentSyncTaskID(taskIDInput)

	var task models.Task
	if res := database.DB.Where("id = ?", taskID).Limit(1).Find(&task); res.Error != nil || res.RowsAffected == 0 {
		fmt.Fprintf(os.Stderr, "错误: 找不到任务 [%s]\n", taskIDInput)
		return
	}

	if task.Type != constant.TaskTypeAgentSyncScript {
		fmt.Fprintf(os.Stderr, "警告: 任务 [%s] 类型为 '%s'，并非 Agent 同步任务类型\n", task.Name, task.Type)
	}

	agentID := ""
	if task.AgentID != nil {
		agentID = *task.AgentID
	}
	syncCfg := task.GetAgentSync()
	if syncCfg != nil && agentID == "" {
		agentID = syncCfg.AgentID
	}

	fmt.Println("========================================")
	fmt.Println("  Agent 脚本/目录同步开始 (Task 模式)  ")
	fmt.Println("========================================")
	fmt.Printf("任务名称: %s (ID: %s)\n", task.Name, task.ID)
	fmt.Printf("目标节点: %s\n", agentID)
	if syncCfg != nil {
		fmt.Printf("映射列表 (%d 个):\n", len(syncCfg.DirMappings))
		for i, m := range syncCfg.DirMappings {
			fmt.Printf("  [%d] %s => %s\n", i+1, m.SourcePath, m.TargetPath)
		}
		if len(syncCfg.IgnoreRules) > 0 {
			fmt.Printf("排除过滤: %s\n", strings.Join(syncCfg.IgnoreRules, ", "))
		}
	}
	fmt.Println("----------------------------------------")

	// 调用内部 API 触发调度执行
	_, err := clibase.CallInternalAPI("POST", "/internal/tasks/execute/"+task.ID, map[string]interface{}{})
	if err != nil {
		fmt.Printf(">> 下发同步任务指令失败: %v\n", err)
		clibase.PrintDBConfigHint("agentsync --task-id " + task.ID)
		return
	}

	fmt.Println(">> 触发指令已成功下发至后台调度器！")
	fmt.Printf(">> 提示: 可以使用 'baihu task status %s' 查看实时下发进度与详细日志。\n", task.ID)
	fmt.Println("========================================")
}

func executeDirectSync(agentID string, mappings []string, cleanTarget bool, ignoreRules string, timeout int) {
	clibase.InitContext(false)

	fmt.Println("========================================")
	fmt.Println("  Agent 脚本/目录直接同步 (Direct 模式)  ")
	fmt.Println("========================================")
	fmt.Printf("目标节点: %s\n", agentID)
	fmt.Printf("清空目标: %t\n", cleanTarget)
	fmt.Printf("映射列表 (%d 个):\n", len(mappings))
	for i, m := range mappings {
		fmt.Printf("  [%d] %s\n", i+1, m)
	}
	if ignoreRules != "" {
		fmt.Printf("排除过滤: %s\n", ignoreRules)
	}
	fmt.Println("----------------------------------------")

	payload := map[string]interface{}{
		"agent_id":     agentID,
		"clean_target": cleanTarget,
		"mappings":     mappings,
		"ignore_rules": strings.Split(ignoreRules, ","),
		"timeout":      timeout,
	}

	bodyBytes, err := clibase.CallInternalAPI("POST", "/internal/agents/sync-direct", payload)
	if err != nil {
		fmt.Printf(">> 直接同步请求失败: %v\n", err)
		clibase.PrintDBConfigHint("agentsync --agent " + agentID + " ...")
		return
	}

	var resp struct {
		Data struct {
			Success bool   `json:"success"`
			Output  string `json:"output"`
			Error   string `json:"error"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &resp); err == nil && resp.Data.Output != "" {
		fmt.Println("[执行详情]")
		fmt.Println(resp.Data.Output)
	} else {
		fmt.Println(string(bodyBytes))
	}

	fmt.Println("========================================")
	fmt.Println("  同步请求已完成  ")
	fmt.Println("========================================")
}

func resolveAgentSyncTaskID(input string) string {
	var t models.Task
	if res := database.DB.Where("id = ?", input).Limit(1).Find(&t); res.Error == nil && res.RowsAffected > 0 {
		return t.ID
	}
	var namedTasks []models.Task
	if res := database.DB.Where("name = ? AND type = ?", input, constant.TaskTypeAgentSyncScript).Find(&namedTasks); res.Error == nil && len(namedTasks) == 1 {
		return namedTasks[0].ID
	}
	if res := database.DB.Where("name LIKE ? AND type = ?", "%"+input+"%", constant.TaskTypeAgentSyncScript).Find(&namedTasks); res.Error == nil && len(namedTasks) == 1 {
		return namedTasks[0].ID
	}
	return input
}
