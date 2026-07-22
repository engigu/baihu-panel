package cmd

import (
	"github.com/engigu/baihu-panel/cmd/builtininstall"
	"github.com/engigu/baihu-panel/cmd/completion"
	"github.com/engigu/baihu-panel/cmd/depinstall"
	"github.com/engigu/baihu-panel/cmd/reposync"
	"github.com/engigu/baihu-panel/cmd/resetpwd"
	"github.com/engigu/baihu-panel/cmd/restore"
	"github.com/engigu/baihu-panel/cmd/task"
	"github.com/engigu/baihu-panel/cmd/version"
	"github.com/engigu/baihu-panel/cmd/webui"
	"github.com/engigu/baihu-panel/internal/bootstrap"
)

// InitHandlers 在 cmd 包内部统一注册所有的子命令 handler
func InitHandlers() {
	// 注册服务后台进程启动命令
	RegisterHandlerWithConfig("server", func(args []string) {
		bootstrap.New().Run()
	}, false)

	// 注册普通 CLI 工具子命令
	RegisterHandler("builtininstall", builtininstall.Run)
	RegisterHandler("completion", completion.Run)
	RegisterHandler("depinstall", depinstall.Run)
	RegisterHandler("reposync", reposync.Run)
	RegisterHandler("resetpwd", resetpwd.Run)
	RegisterHandler("restore", restore.Run)
	RegisterHandler("task", task.Run)
	RegisterHandler("webui", webui.Run)

	// 轻量级命令显式标记 RequireContext = false
	RegisterHandlerWithConfig("version", version.Run, false)
	RegisterHandlerWithConfig("-v", version.Run, false)
	RegisterHandlerWithConfig("-V", version.Run, false)
}
