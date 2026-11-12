package tui

import (
	"sort"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
)

// GetDynamicCommands 获取所有动态命令清单（严格过滤掉 tui 自身，防止嵌套，并追加 exit 退出项）
func GetDynamicCommands() []constant.CommandSpec {
	var list []constant.CommandSpec
	for _, cmd := range constant.Commands {
		if strings.EqualFold(cmd.Name, "tui") {
			continue // 严格禁止嵌套展示 tui
		}
		list = append(list, cmd)
	}
	list = append(list, constant.CommandSpec{
		Name:        "exit",
		Description: "退出白虎面板终端交互控制台",
	})
	return list
}

// FindCommandSpec 根据名称在动态清单中查找命令定义
func FindCommandSpec(name string) *constant.CommandSpec {
	cmds := GetDynamicCommands()
	for i := range cmds {
		if strings.EqualFold(cmds[i].Name, name) {
			return &cmds[i]
		}
	}
	return nil
}

// ExtractUniqueFlags 提取去重后的标准命令行选项（优先保留 '--' 前缀形式）
func ExtractUniqueFlags(flags []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, f := range flags {
		clean := strings.TrimLeft(f, "-")
		if strings.HasPrefix(f, "--") && !seen[clean] {
			seen[clean] = true
			result = append(result, f)
		}
	}

	for _, f := range flags {
		clean := strings.TrimLeft(f, "-")
		if !seen[clean] {
			seen[clean] = true
			result = append(result, f)
		}
	}

	return result
}

// GetSortedSubcommandKeys 获取稳定排序后的子命令键列表
func GetSortedSubcommandKeys(subcmds map[string]string) []string {
	keys := make([]string, 0, len(subcmds))
	for k := range subcmds {
		keys = append(keys, k)
	}

	orderWeight := map[string]int{
		"list": 1, "info": 2, "run": 3, "status": 4, "history": 5,
		"enable": 6, "disable": 7, "apply": 8, "switch": 9, "set": 10,
		"reset": 11, "remove": 12, "delete": 13,
	}

	sort.Slice(keys, func(i, j int) bool {
		wi, oki := orderWeight[keys[i]]
		wj, okj := orderWeight[keys[j]]
		if oki && okj {
			return wi < wj
		}
		if oki {
			return true
		}
		if okj {
			return false
		}
		return keys[i] < keys[j]
	})

	return keys
}
