package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wechat2rss/internal/config"
	"wechat2rss/internal/crawler"
	"wechat2rss/internal/database"
	httpserver "wechat2rss/internal/http"
	"wechat2rss/internal/wechat"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	wechatManager, err := wechat.NewManager(db)
	if err != nil {
		log.Fatalf("wechat manager init: %v", err)
	}

	server := httpserver.New(cfg, db, wechatManager)
	manager := crawler.NewManager(cfg, db)

	crawlerCtx, crawlerCancel := context.WithCancel(context.Background())
	defer crawlerCancel()

	go manager.Start(crawlerCtx)
	go wechatManager.StartPolling(crawlerCtx)

	go func() {
		if err := server.Run(); err != nil {
			log.Fatalf("server run: %v", err)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	crawlerCancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
