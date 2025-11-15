package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"wechat2rss/internal/models"
)

// Connect establishes a PostgreSQL connection using GORM.
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// AutoMigrate runs schema migrations for core models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.WechatSession{},
		&models.Account{},
		&models.Task{},
		&models.TaskLog{},
		&models.Article{},
		&models.Alert{},
	)
}
