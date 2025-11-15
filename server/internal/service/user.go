package service

import (
	"errors"

	"gorm.io/gorm"

	"wechat2rss/internal/models"
)

// EnsureAdmin ensures an admin user exists with provided credentials.
func EnsureAdmin(db *gorm.DB, username, password string) error {
	var existing models.User
	err := db.First(&existing, "username = ?", username).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user := models.User{
			Username:   username,
			ForceReset: true,
		}
		if err := user.SetPassword(password); err != nil {
			return err
		}
		return db.Create(&user).Error
	}
	if err != nil {
		return err
	}
	return nil
}
