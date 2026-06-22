package version

import (
	"fmt"

	"github.com/engigu/baihu-panel/internal/constant"
)

func Run(args []string) {
	fmt.Printf("白虎面板 (Baihu Panel) %s (Build time: %s)\n", constant.Version, constant.BuildTime)
}
