package constant

// CommandSpec 定义了系统级 CLI 命令的统一元数据、子命令及选项接口
type CommandSpec struct {
	Name          string              `json:"name"`
	Description   string              `json:"description"`
	SubCommands   map[string]string   `json:"sub_commands,omitempty"`
	SubFlags      map[string][]string `json:"sub_flags,omitempty"`
	SubPrompts    map[string]string   `json:"sub_prompts,omitempty"`
	SubDefaults   map[string]string   `json:"sub_defaults,omitempty"`
	Flags         []string            `json:"flags,omitempty"`
	Args          []string            `json:"args,omitempty"`
	ArgPrompt     string              `json:"arg_prompt,omitempty"`
	ArgDefault    string              `json:"arg_default,omitempty"`
	ArgHelp       string              `json:"arg_help,omitempty"`
	Confirm       bool                `json:"confirm,omitempty"`
	ConfirmPrompt string              `json:"confirm_prompt,omitempty"`
}

// CommandInfo 别名保持兼容
type CommandInfo = CommandSpec

// Commands 是全系统唯一单源的可用业务命令元数据与参数清单 (Single Source of Truth)
// 包含 CLI 帮助、API 端点、终端面板以及 Shell Tab 自动补全均从此一份数据读取
var Commands = []CommandSpec{
	{
		Name:        "task",
		Description: "系统级任务的列表查询、触发运行、启停控制及状态查看",
		SubCommands: map[string]string{
			"list":    "查询任务列表概览",
			"run":     "立即触发运行任务",
			"status":  "查看任务执行日志和退出码",
			"history": "查看任务近期运行历史记录",
			"enable":  "快速启用指定任务",
			"disable": "快速禁用指定任务",
		},
		SubFlags: map[string][]string{
			"list":    {"--name", "--type", "--page", "--size", "-name", "-type", "-page", "-size"},
			"history": {"--limit", "-limit"},
		},
		SubPrompts: map[string]string{
			"run":     "目标任务ID、名称或 'repo' (输入 b 返回):",
			"status":  "目标任务ID、名称或 'repo' (可追加日志ID，输入 b 返回):",
			"history": "目标任务ID、名称或 'repo' (输入 b 返回):",
			"enable":  "要启用的任务ID、名称或 'repo' (输入 b 返回):",
			"disable": "要禁用的任务ID、名称或 'repo' (输入 b 返回):",
			"list":    "可选过滤参数 (例如: -name 签到 -page 1 -size 20，直接回车跳过):",
		},
	},
	{
		Name:        "app",
		Description: "白虎声明式应用的解析、安装部署、场景切换与生命周期管理",
		SubCommands: map[string]string{
			"apply":  "解析并应用本地或远程 app.yaml",
			"list":   "查看已安装的应用列表",
			"info":   "查看指定应用的详细信息",
			"switch": "一键切换应用的运行场景预设",
			"remove": "卸载应用并清理受控任务",
		},
		SubFlags: map[string][]string{
			"apply":  {"--scenario", "--skip-setup", "--skip-sync", "--clean-data"},
			"switch": {"--scenario"},
			"remove": {"--clean-data"},
		},
		SubPrompts: map[string]string{
			"apply":  "app.yaml 本地文件路径或远程 URL (可追加 --scenario 等):",
			"info":   "目标应用ID或应用标识名 (输入 b 返回):",
			"switch": "应用ID或应用标识名 (可追加 --scenario):",
			"remove": "要卸载的应用ID或标识名 (可追加 --clean-data):",
		},
		Flags: []string{"--scenario", "--skip-setup", "--skip-sync", "--clean-data"},
	},
	{
		Name:        "reposync",
		Description: "同步远程 Git 仓库或文件到本地目录",
		ArgPrompt:   "reposync 选项参数 (直接回车执行默认向导/全量同步):",
		Flags: []string{
			"--source-type", "--source-url", "--target-path", "--branch",
			"--path", "--single-file", "--proxy", "--proxy-url",
			"--auth-token", "--http-proxy", "--whitelist-paths",
			"--blacklist", "--dependence", "--extensions", "--commenttotask",
			"--pre-command", "--post-command", "--repo-name",
			"--task-id", "--task-timeout", "--task-langs",
		},
	},
	{
		Name:        "depinstall",
		Description: "一键补全指定任务日志中的缺失依赖包",
		ArgPrompt:   "报错的任务日志ID (Log ID):",
		ArgHelp:     "输入任务运行流水产生的 Log ID，系统将自动分析缺失依赖并安装",
	},
	{
		Name:        "agentsync",
		Description: "同步本地脚本与目录至远程 Agent 节点",
		ArgPrompt:   "agentsync 选项参数 (例如: --agent default，直接回车执行默认):",
		Flags: []string{
			"--task-id", "--agent", "--mapping", "--mappings", "--clean-target", "--ignore", "--timeout",
		},
	},
	{
		Name:          "server",
		Description:   "启动面板后台服务进程",
		Confirm:       true,
		ConfirmPrompt: "启动 server 命令将常驻占用终端并开始监听服务端口，按 Ctrl+C 可停止。是否确认启动?",
	},
	{
		Name:        "webui",
		Description: "白虎面板前端 WebUI 资源包的管理、激活与回退",
		SubCommands: map[string]string{
			"list":   "列出当前安装的所有前端资源包",
			"set":    "设置激活指定的 WebUI",
			"reset":  "一键回退到系统默认的内置 WebUI",
			"delete": "删除指定的 WebUI 资源包",
		},
		SubPrompts: map[string]string{
			"set":    "要激活的 WebUI 标识名称 (输入 b 返回):",
			"delete": "要删除的 WebUI 标识名称 (输入 b 返回):",
		},
	},
	{
		Name:          "mcp",
		Description:   "启动白虎面板 Model Context Protocol (MCP) 标准输入输出服务",
		Confirm:       true,
		ConfirmPrompt: "MCP 服务通过标准 Stdio 通道进行 JSON-RPC 通信，通常供 AI 客户端调用。是否确认启动?",
	},
	{
		Name:        "dropcache",
		Description: "释放指定目录或文件的只读 Page Cache 缓存",
		ArgPrompt:   "要释放 PageCache 的目录或文件路径:",
		ArgDefault:  ".",
		ArgHelp:     "默认路径为当前目录 .，回车执行清理",
	},
	{
		Name:          "builtininstall",
		Description:   "为所有 mise 管理的 Node.js 和 Python 环境安装内建助手库",
		Confirm:       true,
		ConfirmPrompt: "此操作将为 mise 管理的所有 Node.js 和 Python 环境安装内建助手库，是否继续?",
	},
	{
		Name:        "resetpwd",
		Description: "交互式重置 admin 管理员账号密码",
		ArgPrompt:   "要重置密码的管理员用户名:",
		ArgDefault:  "admin",
		ArgHelp:     "默认用户名 admin，留空直接回车将使用默认值",
	},
	{
		Name:        "restore",
		Description: "从本地 zip 备份包全量恢复系统数据",
		ArgPrompt:   "本地备份文件路径 (.zip):",
		ArgHelp:     "请输入存在的备份 zip 压缩包路径，回车执行",
	},
	{
		Name:        "completion",
		Description: "生成当前 Shell (PowerShell/Bash/Zsh) 的 Tab 自动补全脚本",
		Args:        []string{"powershell", "pwsh", "bash", "zsh"},
	},
	{
		Name:        "tui",
		Description: "启动白虎面板交互式终端管理控制台 (TUI)",
	},
	{
		Name:        "version",
		Description: "查看当前系统版本号 (同 -v, -V)",
	},
}
