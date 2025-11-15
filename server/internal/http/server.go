package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wechat2rss/internal/config"
	"wechat2rss/internal/models"
	"wechat2rss/internal/service"
	"wechat2rss/internal/wechat"
)

const sessionName = "wechat2rss_session"

// Server wires routing, middleware, and HTTP server.
type Server struct {
	cfg    *config.Config
	db     *gorm.DB
	engine *gin.Engine
	http   *http.Server
	wechat *wechat.Manager
}

// New constructs the HTTP server and routes.
func New(cfg *config.Config, db *gorm.DB, wm *wechat.Manager) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors.Default())

	store := cookie.NewStore([]byte(cfg.SessionSecret))
	router.Use(sessions.Sessions(sessionName, store))

	s := &Server{
		cfg:    cfg,
		db:     db,
		engine: router,
		wechat: wm,
	}

	if err := service.EnsureAdmin(db, cfg.AdminUser, cfg.AdminPassword); err != nil {
		panic(fmt.Sprintf("ensure admin: %v", err))
	}

	s.registerRoutes()

	s.http = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}
	return s
}

func (s *Server) registerRoutes() {
	s.engine.GET("/health", s.handleHealth)
	s.engine.GET("/feed/:id", s.handleFeed)

	api := s.engine.Group("/api")
	{
		api.POST("/login", s.handleLogin)

		auth := api.Group("/")
		auth.Use(s.requireSession)
		{
			auth.GET("/me", s.handleMe)
			auth.POST("/logout", s.handleLogout)
			auth.POST("/password", s.handlePasswordUpdate)
		}

		secured := api.Group("/")
		secured.Use(s.requireSession, s.requirePasswordSet)
		{
			secured.GET("/accounts", s.handleListAccounts)
			secured.POST("/accounts", s.handleCreateAccount)
			secured.GET("/accounts/:id", s.handleGetAccount)
			secured.PUT("/accounts/:id", s.handleUpdateAccount)
			secured.DELETE("/accounts/:id", s.handleDeleteAccount)

			secured.POST("/accounts/:id/tasks", s.handleCreateTask)
			secured.GET("/accounts/:id/articles", s.handleListArticles)
			secured.GET("/tasks", s.handleListTasks)
			secured.GET("/tasks/:id/logs", s.handleTaskLogs)

			secured.GET("/wechat/sessions", s.handleListWechatSessions)
			secured.POST("/wechat/sessions", s.handleCreateWechatSession)
			secured.GET("/wechat/sessions/:id", s.handleGetWechatSession)
			secured.GET("/wechat/search", s.handleWechatSearch)
		}
	}

	s.registerStaticRoutes()
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	return s.http.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s *Server) handleHealth(c *gin.Context) {
	respondOK(c, apiData{"status": "ok", "time": time.Now().UTC()})
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *Server) handleLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	var user models.User
	if err := s.db.First(&user, "username = ?", req.Username).Error; err != nil {
		respondError(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if !user.CheckPassword(req.Password) {
		respondError(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	if err := session.Save(); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to persist session")
		return
	}

	respondOK(c, apiData{
		"id":            user.ID,
		"username":      user.Username,
		"require_reset": user.ForceReset,
	})
}

func (s *Server) requireSession(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("user_id") == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		c.Abort()
		return
	}
	c.Set("user_id", session.Get("user_id"))
	c.Next()
}

func (s *Server) requirePasswordSet(c *gin.Context) {
	user, err := s.currentUser(c)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "invalid session")
		c.Abort()
		return
	}
	if user.ForceReset {
		respondError(c, http.StatusForbidden, "password reset required")
		c.Abort()
		return
	}
	c.Next()
}

func (s *Server) handleMe(c *gin.Context) {
	user, err := s.currentUser(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusUnauthorized, "invalid session")
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to load user")
		return
	}

	respondOK(c, apiData{
		"id":            user.ID,
		"username":      user.Username,
		"require_reset": user.ForceReset,
		"created_at":    user.CreatedAt,
	})
}

func (s *Server) handleLogout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	if err := session.Save(); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to clear session")
		return
	}
	respondOK(c, apiData{"status": "logged_out"})
}

type passwordUpdateRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

func (s *Server) handlePasswordUpdate(c *gin.Context) {
	var req passwordUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	user, err := s.currentUser(c)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "invalid session")
		return
	}

	if !user.CheckPassword(req.OldPassword) {
		respondError(c, http.StatusForbidden, "old password incorrect")
		return
	}

	if err := user.SetPassword(req.NewPassword); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update password")
		return
	}
	user.ForceReset = false

	if err := s.db.Save(user).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to save user")
		return
	}

	respondOK(c, apiData{"status": "password_updated"})
}

func (s *Server) registerStaticRoutes() {
	if s.cfg.StaticDir == "" {
		return
	}
	staticDir := s.cfg.StaticDir
	s.engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "" {
			path = "/"
		}
		clean := filepath.Clean(path)
		target := filepath.Join(staticDir, clean)
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			c.File(target)
			return
		}
		index := filepath.Join(staticDir, "index.html")
		if _, err := os.Stat(index); err == nil {
			c.File(index)
			return
		}
		c.Status(http.StatusNotFound)
	})
}

func (s *Server) currentUser(c *gin.Context) (*models.User, error) {
	idVal, exists := c.Get("user_id")
	if !exists {
		session := sessions.Default(c)
		idVal = session.Get("user_id")
		if idVal == nil {
			return nil, gorm.ErrRecordNotFound
		}
	}

	var user models.User
	if err := s.db.First(&user, "id = ?", idVal).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
