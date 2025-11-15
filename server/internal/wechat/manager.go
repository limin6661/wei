package wechat

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"wechat2rss/internal/models"
	"wechat2rss/internal/util"
)

// Manager handles wechat login sessions lifecycle.
type Manager struct {
	db           *gorm.DB
	client       *MPClient
	pollInterval time.Duration
}

// NewManager creates a manager with default poll interval.
func NewManager(db *gorm.DB) (*Manager, error) {
	client, err := NewMPClient()
	if err != nil {
		return nil, err
	}
	return &Manager{
		db:           db,
		client:       client,
		pollInterval: 2 * time.Second,
	}, nil
}

// CreateSession generates QR code and persists session record.
func (m *Manager) CreateSession(ctx context.Context) (*models.WechatSession, error) {
	qr, err := m.client.FetchQRCode(ctx)
	if err != nil {
		return nil, err
	}
	key, err := util.RandHex(8)
	if err != nil {
		return nil, err
	}
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(qr.Data)
	session := models.WechatSession{
		SessionKey: key,
		UUID:       qr.UUID,
		QRCode:     dataURI,
		Status:     models.SessionStatusPending,
	}
	if err := m.db.Create(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// StartPolling monitors all pending sessions to update their status.
func (m *Manager) StartPolling(ctx context.Context) {
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.pollSessions(ctx); err != nil {
				log.Printf("wechat poll error: %v", err)
			}
		}
	}
}

func (m *Manager) pollSessions(ctx context.Context) error {
	var sessions []models.WechatSession
	if err := m.db.Where("status IN ?", []string{models.SessionStatusPending, models.SessionStatusScanning}).
		Limit(20).
		Find(&sessions).Error; err != nil {
		return err
	}

	for _, session := range sessions {
		if err := m.updateSession(ctx, &session); err != nil {
			log.Printf("update session %d error: %v", session.ID, err)
		}
	}
	return nil
}

func (m *Manager) updateSession(ctx context.Context, session *models.WechatSession) error {
	status, err := m.client.AskStatus(ctx, session.UUID)
	if err != nil {
		return err
	}

	now := time.Now()
	switch status.State {
	case "waiting":
		return m.db.Model(session).Updates(map[string]any{
			"status":    models.SessionStatusPending,
			"last_ping": &now,
		}).Error
	case "scanned":
		return m.db.Model(session).Updates(map[string]any{
			"status":    models.SessionStatusScanning,
			"last_ping": &now,
		}).Error
	case "authorized":
		cookies, token, err := m.client.FinalizeLogin(ctx, status.RedirectURL)
		if err != nil {
			return fmt.Errorf("finalize login: %w", err)
		}
		expiry := now.Add(12 * time.Hour)
		return m.db.Model(session).Updates(map[string]any{
			"status":     models.SessionStatusActive,
			"cookie":     cookies,
			"token":      token,
			"last_ping":  &now,
			"expires_at": &expiry,
		}).Error
	case "expired":
		return m.db.Model(session).Updates(map[string]any{
			"status": models.SessionStatusExpired,
		}).Error
	default:
		return nil
	}
}
