package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User represents admin account.
type User struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex"`
	PasswordHash string
	ForceReset   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// SetPassword hashes and stores a password.
func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword verifies the password.
func (u *User) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
}

// WechatSession tracks an authenticated session.
type WechatSession struct {
	ID         uint   `gorm:"primaryKey"`
	SessionKey string `gorm:"uniqueIndex"`
	UUID       string `gorm:"index"`
	QRCode     string `gorm:"type:text"`
	Cookie     string `gorm:"type:text"`
	Token      string `gorm:"type:text"`
	Status     string `gorm:"index"` // pending, scanning, active, expired
	ExpiresAt  *time.Time
	LastPing   *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Account represents a tracked public account.
type Account struct {
	ID         uint `gorm:"primaryKey"`
	Name       string
	WechatID   string `gorm:"uniqueIndex"`
	BizID      string `gorm:"index"`
	Alias      string
	Status     string `gorm:"default:'active'"`
	SessionID  *uint
	Session    *WechatSession `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	LastTaskID *uint
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Task records a crawl execution.
type Task struct {
	ID         uint    `gorm:"primaryKey"`
	AccountID  uint    `gorm:"index"`
	Account    Account `gorm:"constraint:OnDelete:CASCADE"`
	Status     string  `gorm:"index"` // pending, running, success, failed
	RetryCount int
	ErrorMsg   string `gorm:"type:text"`
	StartedAt  *time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type TaskLog struct {
	ID        uint      `gorm:"primaryKey"`
	TaskID    uint      `gorm:"index"`
	Level     string    `gorm:"default:'info'"`
	Message   string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"index"`
}

const (
	TaskStatusPending = "pending"
	TaskStatusRunning = "running"
	TaskStatusSuccess = "success"
	TaskStatusFailed  = "failed"

	TaskMaxRetries = 3
)

const (
	SessionStatusPending  = "pending"
	SessionStatusScanning = "scanning"
	SessionStatusActive   = "active"
	SessionStatusExpired  = "expired"
)

// Article stores fetched items.
type Article struct {
	ID              uint   `gorm:"primaryKey"`
	AccountID       uint   `gorm:"index"`
	WechatArticleID string `gorm:"index"`
	Title           string
	Summary         string `gorm:"type:text"`
	ContentHTML     string `gorm:"type:text"`
	RawURL          string
	PublishedAt     time.Time `gorm:"index"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Alert records system alerts.
type Alert struct {
	ID         uint   `gorm:"primaryKey"`
	Type       string `gorm:"index"`
	Status     string
	Payload    string `gorm:"type:text"`
	NotifiedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
