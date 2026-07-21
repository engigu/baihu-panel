package cmd

// CommandSpec 定义了系统级 CLI 命令的统一元数据、子命令及选项接口
type CommandSpec struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	SubCommands map[string]string `json:"sub_commands,omitempty"`
	Flags       []string          `json:"flags,omitempty"`
	Args        []string          `json:"args,omitempty"`
}

// CommandInfo 别名保持兼容
type CommandInfo = CommandSpec

// Commands 是全系统唯一单源的可用业务命令元数据与参数清单 (Single Source of Truth)
// 包含 CLI 帮助、API 端点、终端面板以及 Shell Tab 自动补全均从此一份数据读取
var Commands = []CommandSpec{
	{
		Name:        "server",
		Description: "启动面板后台服务进程",
	},
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
	},
	{
		Name:        "reposync",
		Description: "同步远程 Git 仓库或文件到本地目录",
		Flags: []string{
			"--source-type", "--source-url", "--target-path", "--branch",
			"--path", "--single-file", "--proxy", "--proxy-url",
			"--auth-token", "--http-proxy", "--whitelist-paths",
			"--blacklist", "--dependence", "--extensions", "--commenttotask",
		},
	},
	{
		Name:        "resetpwd",
		Description: "交互式重置 admin 管理员账号密码",
	},
	{
		Name:        "restore",
		Description: "从本地 zip 备份包全量恢复系统数据",
	},
	{
		Name:        "builtininstall",
		Description: "为所有 mise 管理的 Node.js 和 Python 环境安装内建助手库",
	},
	{
		Name:        "depinstall",
		Description: "一键补全指定任务日志中的缺失依赖包",
	},
	{
		Name:        "version",
		Description: "查看当前系统版本号 (同 -v, -V)",
	},
	{
		Name:        "completion",
		Description: "生成当前 Shell (PowerShell/Bash/Zsh) 的 Tab 自动补全脚本",
		Args:        []string{"powershell", "bash", "zsh"},
	},
}

// CommandHandler 定义命令执行函数
type CommandHandler func(args []string)

// Handlers 维护除了 server 之外的命令的执行入口
var Handlers = map[string]CommandHandler{}

// RegisterHandler 供各个子命令包注册自己的处理函数
func RegisterHandler(name string, handler CommandHandler) {
	Handlers[name] = handler
}
