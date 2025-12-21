package bootstrap

import (
	"fmt"
	"os"

	"baihu/internal/constant"
	"baihu/internal/database"
	"baihu/internal/logger"
	"baihu/internal/router"
	"baihu/internal/services"

	"github.com/gin-gonic/gin"
)

type App struct {
	Config *services.AppConfig
	Router *gin.Engine
}

func New() *App {
	app := &App{}
	app.initConfig()
	app.initDatabase()
	app.initRouter()
	return app
}

func (a *App) initConfig() {
	cfg, err := services.LoadConfig(constant.ConfigPath)
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
	}

	if err := database.Init(dbCfg); err != nil {
		logger.Fatalf("Failed to init database: %v", err)
	}

	if err := database.Migrate(); err != nil {
		logger.Fatalf("Failed to migrate database: %v", err)
	}
}

func (a *App) initRouter() {
	ctrls := router.RegisterControllers()
	a.Router = router.Setup(ctrls)
}

func (a *App) Run() {
	addr := fmt.Sprintf("%s:%d", a.Config.Server.Host, a.Config.Server.Port)
	logger.Infof("Starting server on %s", addr)
	a.Router.Run(addr)
}
