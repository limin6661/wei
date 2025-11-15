package http

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"wechat2rss/internal/models"
	"wechat2rss/internal/wechat"
)

func (s *Server) handleListWechatSessions(c *gin.Context) {
	var sessions []models.WechatSession
	if err := s.db.Order("id desc").Limit(20).Find(&sessions).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to list sessions")
		return
	}

	var result []wechatSessionView
	for i := range sessions {
		result = append(result, toWechatSessionView(&sessions[i]))
	}
	respondOK(c, apiData{"sessions": result})
}

func (s *Server) handleCreateWechatSession(c *gin.Context) {
	if s.wechat == nil {
		respondError(c, http.StatusInternalServerError, "wechat manager unavailable")
		return
	}
	session, err := s.wechat.CreateSession(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, fmt.Sprintf("failed to create session: %v", err))
		return
	}
	respondOK(c, apiData{"session": toWechatSessionView(session)})
}

func (s *Server) handleGetWechatSession(c *gin.Context) {
	session, err := s.findWechatSession(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusNotFound, "session not found")
		return
	}
	respondOK(c, apiData{"session": toWechatSessionView(session)})
}

func (s *Server) findWechatSession(idParam string) (*models.WechatSession, error) {
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return nil, err
	}
	var session models.WechatSession
	if err := s.db.First(&session, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

type wechatSessionView struct {
	ID         uint       `json:"id"`
	SessionKey string     `json:"session_key"`
	Status     string     `json:"status"`
	QRCode     string     `json:"qr_code"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastPing   *time.Time `json:"last_ping"`
	CreatedAt  time.Time  `json:"created_at"`
}

func toWechatSessionView(ses *models.WechatSession) wechatSessionView {
	return wechatSessionView{
		ID:         ses.ID,
		SessionKey: ses.SessionKey,
		Status:     ses.Status,
		QRCode:     ses.QRCode,
		ExpiresAt:  ses.ExpiresAt,
		LastPing:   ses.LastPing,
		CreatedAt:  ses.CreatedAt,
	}
}

func (s *Server) handleWechatSearch(c *gin.Context) {
	sessionID := c.Query("session_id")
	query := c.Query("query")
	if sessionID == "" || query == "" {
		respondError(c, http.StatusBadRequest, "session_id and query required")
		return
	}
	id, err := strconv.Atoi(sessionID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid session_id")
		return
	}
	var session models.WechatSession
	if err := s.db.First(&session, "id = ?", id).Error; err != nil {
		respondError(c, http.StatusNotFound, "session not found")
		return
	}
	if session.Status != models.SessionStatusActive {
		respondError(c, http.StatusBadRequest, "session not active")
		return
	}
	results, err := wechat.SearchAccounts(c.Request.Context(), wechat.Credentials{
		Cookie: session.Cookie,
		Token:  session.Token,
	}, query, 0)
	if err != nil {
		respondError(c, http.StatusInternalServerError, fmt.Sprintf("search failed: %v", err))
		return
	}
	respondOK(c, apiData{"results": results})
}
