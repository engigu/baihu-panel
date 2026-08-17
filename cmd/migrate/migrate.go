package migrate

import (
	"fmt"
	"github.com/engigu/baihu-panel/internal/bootstrap"
	"github.com/engigu/baihu-panel/internal/services"
)

func Run(args []string) {
	fmt.Println("Starting Migration V3...")
	// 初始化基础环境（配置和数据库，但不运行常规 Migrate，因为我们想手动控）
	// 不过 bootstrap.New() 会调用 Migrate().
	// 我们可以调用 InitBasic()
	app := bootstrap.InitBasicForCmd()
	if app == nil {
		fmt.Println("Failed to initialize app")
		return
	}

	// 此时数据库已经连接，Migrate() 已经运行过了（因为 bootstrap.InitBasic 调用了 app.initDatabase）
	// 由于我们在 Migrate() 中集成了 RunMigrationV3()，所以其实已经跑过了。
	// 如果用户想重复跑，或者单独跑：
	err := services.RunMigrationV3()
	if err != nil {
		fmt.Printf("Migration failed: %v\n", err)
		return
	}
	fmt.Println("Migration V3 completed successfully.")
}
