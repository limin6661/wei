package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	Port              int
	DatabaseURL       string
	SessionSecret     string
	AdminUser         string
	AdminPassword     string
	ChromiumPath      string
	CrawlerConcurrent int
	TaskPollInterval  int
	StaticDir         string
}

// Load reads environment variables (populating defaults) and returns Config.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:              getInt("APP_PORT", 8080),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		SessionSecret:     getStr("SESSION_SECRET", "wechat2rss-secret"),
		AdminUser:         getStr("ADMIN_USER", "admin"),
		AdminPassword:     getStr("ADMIN_PASSWORD", "changeme"),
		ChromiumPath:      os.Getenv("CHROMIUM_PATH"),
		CrawlerConcurrent: getInt("CRAWLER_CONCURRENCY", 1),
		TaskPollInterval:  getInt("TASK_POLL_INTERVAL", 5),
		StaticDir:         os.Getenv("WEB_STATIC_DIR"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}
