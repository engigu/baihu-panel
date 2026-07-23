package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/executor"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/router"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/tunnel"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/engigu/baihu-panel/internal/windows"

	"github.com/gin-gonic/gin"
)

type App struct {
	Config *services.AppConfig
	Router *gin.Engine
}

func New() *App {
	app := InitBasic()
	app.initRouter()
	
	// 初始化完成后将路由引擎注入到隧道模块，以支持高性能的纯内存代理
	tunnel.SetLocalEngine(app.Router)
	
	// 初始化隧道后台服务 (读取配置决定角色并启动服务)
	tunnel.Init()
	
	// 启动系统级后台定时任务调度器
	executor.InitSysCron()

	// 初始化完成后回收一次内存
	utils.FreeMemory()

	return app
}

var (
	globalApp *App
	initOnce  sync.Once
)

func InitBasic() *App {
	initOnce.Do(func() {
		// Windows 平台下强校验 pwsh.exe 是否存在
		windows.VerifyPwsh()

		app := &App{}
		utils.InitRuntime()
		utils.InitSecretKey()

		// 自动加载配置 (内部会自动处理 BH_CONFIG_PATH 环境变量与默认路径的优先级)
		app.initConfigWithPath("")
		app.initDatabase()
		
		logger.Infof("[System] 低于1.0.11版本升级最新版本错误指引: https://github.com/engigu/baihu-panel/issues/64")
		globalApp = app
	})
	return globalApp
}

// InitBasicForCmd 专为命令行工具定制的基础环境初始化入口
// 内部会调高控制台日志过滤级别以自动静默屏蔽刷屏的底层系统与组件启动 Info 日志
func InitBasicForCmd() *App {
	logger.SetLevel("warn")
	return InitBasic()
}

// func (a *App) initConfig() {
// 	a.initConfigWithPath(constant.ConfigPath)
// }

func (a *App) initConfigWithPath(path string) {
	cfg, err := services.LoadConfig(path)
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}
	a.Config = cfg

	// Ensure directories exist
	err = os.MkdirAll(constant.DataDir, 0755)
	if err != nil {
		return
	}
	err = os.MkdirAll(constant.ScriptsWorkDir, 0755)
	if err != nil {
		return
	}

	a.setupBaihuBin()
}

func (a *App) setupBaihuBin() {
	binDir := filepath.Join(constant.DataDir, "bin")
	_ = os.MkdirAll(binDir, 0755)

	exe, err := os.Executable()
	if err == nil {
		linkPath := filepath.Join(binDir, "baihu")
		linkPath += windows.GetExeExtension()
		os.Remove(linkPath)
		_ = os.Symlink(exe, linkPath)
	}
}

func (a *App) initDatabase() {
	dbCfg := &database.Config{
		Type:     a.Config.Database.Type,
		Host:     a.Config.Database.Host,
		Port:     a.Config.Database.Port,
		User:     a.Config.Database.User,
		Password: a.Config.Database.Password,
		DBName:   a.Config.Database.DBName,
		Path:     a.Config.Database.Path,
		DSN:      a.Config.Database.DSN,
		SSLMode:  a.Config.Database.SSLMode,
	}

	if err := database.Init(dbCfg); err != nil {
		logger.Fatalf("Failed to init database: %v", err)
	}

	// 记录各个初始化阶段的时间
	startTime := time.Now()

	// 执行 V3 迁移（ID 变更迁移）
	if err := services.RunMigrationV3(); err != nil {
		logger.Fatalf("Failed to run V3 migration: %v", err)
	}
	v3Duration := time.Since(startTime)
	logger.Infof("[Database] V3 迁移检查完成, 耗时: %v", v3Duration)

	// 执行表结构同步
	migrateStart := time.Now()
	if err := database.Migrate(); err != nil {
		logger.Fatalf("Failed to migrate database: %v", err)
	}
	migrateDuration := time.Since(migrateStart)
	logger.Infof("[Database] 表结构同步完成, 耗时: %v", migrateDuration)
	logger.Infof("[Database] 数据库总初始化耗时: %v", time.Since(startTime))
}

func (a *App) initRouter() {
	ctrls := router.RegisterControllers()
	a.Router = router.Setup(ctrls)
}

func (a *App) Run() {
	addr := fmt.Sprintf("%s:%d", a.Config.Server.Host, a.Config.Server.Port)
	logger.Infof("[HTTP] 服务正在启动，监听地址: http://%s", addr)
	if err := a.Router.Run(addr); err != nil {
		logger.Fatalf("[HTTP] 服务启动失败: %v", err)
	}
}
