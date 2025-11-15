package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"wechat2rss/internal/models"
)

type accountRequest struct {
	Name      string `json:"name" binding:"required"`
	WechatID  string `json:"wechat_id" binding:"required"`
	BizID     string `json:"biz_id"`
	Alias     string `json:"alias"`
	Status    string `json:"status"`
	SessionID *uint  `json:"session_id"`
}

func (s *Server) handleCreateAccount(c *gin.Context) {
	var req accountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	account := models.Account{
		Name:      req.Name,
		WechatID:  req.WechatID,
		BizID:     req.BizID,
		Alias:     req.Alias,
		Status:    defaultStatus(req.Status),
		SessionID: req.SessionID,
	}

	if err := s.db.Create(&account).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create account")
		return
	}

	respondOK(c, apiData{"account": toAccountView(&account)})
}

func (s *Server) handleListAccounts(c *gin.Context) {
	var accounts []models.Account
	if err := s.db.Order("id desc").Find(&accounts).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to list accounts")
		return
	}
	var result []accountView
	for i := range accounts {
		result = append(result, *toAccountView(&accounts[i]))
	}
	respondOK(c, apiData{"accounts": result})
}

func (s *Server) handleGetAccount(c *gin.Context) {
	account, err := s.findAccount(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusNotFound, "account not found")
		return
	}
	respondOK(c, apiData{"account": toAccountView(account)})
}

func (s *Server) handleUpdateAccount(c *gin.Context) {
	account, err := s.findAccount(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusNotFound, "account not found")
		return
	}

	var req accountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	account.Name = req.Name
	account.WechatID = req.WechatID
	account.BizID = req.BizID
	account.Alias = req.Alias
	account.Status = defaultStatus(req.Status)
	account.SessionID = req.SessionID

	if err := s.db.Save(account).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update account")
		return
	}

	respondOK(c, apiData{"account": toAccountView(account)})
}

func (s *Server) handleDeleteAccount(c *gin.Context) {
	account, err := s.findAccount(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusNotFound, "account not found")
		return
	}

	if err := s.db.Delete(account).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to delete account")
		return
	}

	respondOK(c, apiData{"deleted": account.ID})
}

func (s *Server) findAccount(idParam string) (*models.Account, error) {
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return nil, err
	}
	var account models.Account
	if err := s.db.First(&account, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func defaultStatus(value string) string {
	if value == "" {
		return "active"
	}
	return value
}

type accountView struct {
	ID         uint      `json:"id"`
	Name       string    `json:"name"`
	WechatID   string    `json:"wechat_id"`
	BizID      string    `json:"biz_id"`
	Alias      string    `json:"alias"`
	Status     string    `json:"status"`
	SessionID  *uint     `json:"session_id"`
	LastTaskID *uint     `json:"last_task_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func toAccountView(a *models.Account) *accountView {
	return &accountView{
		ID:         a.ID,
		Name:       a.Name,
		WechatID:   a.WechatID,
		BizID:      a.BizID,
		Alias:      a.Alias,
		Status:     a.Status,
		SessionID:  a.SessionID,
		LastTaskID: a.LastTaskID,
		CreatedAt:  a.CreatedAt,
		UpdatedAt:  a.UpdatedAt,
	}
}
