package dropcache

import (
	"fmt"
	"os"

	"github.com/engigu/baihu-panel/internal/memopt"
)

// Run 执行 dropcache 命令，回收指定文件或目录的只读 Page Cache
func Run(args []string) {
	if len(args) == 0 {
		fmt.Println("用法: baihu dropcache <路径1> [路径2...]")
		os.Exit(0)
	}

	count, _ := memopt.DropCache(args...)
	fmt.Printf("[MemOpt] 成功释放 %d 个文件的 Page Cache\n", count)
}
