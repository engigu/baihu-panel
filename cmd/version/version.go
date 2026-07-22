package version

import (
	"fmt"

	"github.com/engigu/baihu-panel/internal/constant"
)

func Run(args []string) {
	fmt.Printf("baihu-panel %s (Build time: %s)\n", constant.Version, constant.BuildTime)
}
