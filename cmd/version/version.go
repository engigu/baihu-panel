package version

import (
	"fmt"

	"github.com/engigu/baihu-panel/cmd"
	"github.com/engigu/baihu-panel/internal/constant"
)

func init() {
	cmd.RegisterHandler("version", Run)
	cmd.RegisterHandler("-v", Run)
	cmd.RegisterHandler("-V", Run)
}

func Run(args []string) {
	fmt.Printf("baihu-panel %s (Build time: %s)\n", constant.Version, constant.BuildTime)
}
