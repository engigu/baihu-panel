package database

import (
	"baihu/internal/models"
)

func Migrate() error {
	return AutoMigrate(
		&models.User{},
		&models.Task{},
		&models.TaskLog{},
		&models.Script{},
		&models.EnvironmentVariable{},
		&models.Setting{},
		&models.LoginLog{},
	)
}
