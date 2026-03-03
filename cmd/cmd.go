package cmd

import (
	"github.com/engigu/baihu-panel/cmd/reposync"
	"github.com/engigu/baihu-panel/cmd/resetpwd"
	"github.com/engigu/baihu-panel/cmd/restore"
	// "github.com/engigu/baihu-panel/cmd/migrate"
)

// CommandHandler 定义命令执行函数
type CommandHandler func(args []string)

// Handlers 维护了除了 server 之外的命令的执行入口
var Handlers = map[string]CommandHandler{
	"reposync": reposync.Run,
	"resetpwd": resetpwd.Run,
	"restore":  restore.Run,
	// "migrate":  migrate.Run,
}
