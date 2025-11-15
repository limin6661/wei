package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"wechat2rss/internal/models"
)

func (s *Server) handleCreateTask(c *gin.Context) {
	account, err := s.findAccount(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusNotFound, "account not found")
		return
	}

	task := models.Task{
		AccountID: account.ID,
		Status:    models.TaskStatusPending,
	}

	if err := s.db.Create(&task).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create task")
		return
	}

	if err := s.db.Model(account).Update("last_task_id", task.ID).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update account task")
		return
	}

	respondOK(c, apiData{
		"task": task,
	})
}

func (s *Server) handleListTasks(c *gin.Context) {
	var tasks []models.Task
	if err := s.db.Preload("Account").Order("id desc").Limit(100).Find(&tasks).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	type taskView struct {
		ID         uint         `json:"id"`
		AccountID  uint         `json:"account_id"`
		Account    *accountView `json:"account,omitempty"`
		Status     string       `json:"status"`
		ErrorMsg   string       `json:"error"`
		StartedAt  *time.Time   `json:"started_at"`
		FinishedAt *time.Time   `json:"finished_at"`
		CreatedAt  time.Time    `json:"created_at"`
	}

	var result []taskView
	for _, t := range tasks {
		var accView *accountView
		if t.Account.ID != 0 {
			accView = &accountView{
				ID:       t.Account.ID,
				Name:     t.Account.Name,
				WechatID: t.Account.WechatID,
				Status:   t.Account.Status,
			}
		}
		result = append(result, taskView{
			ID:         t.ID,
			AccountID:  t.AccountID,
			Account:    accView,
			Status:     t.Status,
			ErrorMsg:   t.ErrorMsg,
			StartedAt:  t.StartedAt,
			FinishedAt: t.FinishedAt,
			CreatedAt:  t.CreatedAt,
		})
	}

	respondOK(c, apiData{"tasks": result})
}

func (s *Server) handleTaskLogs(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid id")
		return
	}

	var logs []models.TaskLog
	if err := s.db.Where("task_id = ?", id).Order("created_at asc").Limit(100).Find(&logs).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to load logs")
		return
	}

	respondOK(c, apiData{"logs": logs})
}
