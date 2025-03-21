package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stellarfrp/config"
	"stellarfrp/internal/api"
	"stellarfrp/pkg/database"
	"stellarfrp/pkg/geetest"
	"stellarfrp/pkg/logger"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 初始化日志
	logger := logger.NewLogger(cfg.LogLevel)
	defer logger.Sync()

	// 初始化数据库连接
	db, err := database.NewMySQLConnection(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", err)
	}
	defer db.Close()

	// 初始化Redis连接
	redisClient, err := database.NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", err)
	}
	defer redisClient.Close()

	// 初始化极验验证客户端
	geetestClient := geetest.NewGeetestClient(
		cfg.Geetest.CaptchaID,
		cfg.Geetest.CaptchaKey,
		cfg.Geetest.APIServer,
	)

	// 初始化API路由
	router := api.SetupRouter(cfg, logger, db, redisClient, geetestClient)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.APIPort),
		Handler: router,
	}

	// 启动服务器（非阻塞）
	go func() {
		logger.Info(fmt.Sprintf("Server is running on port %d", cfg.APIPort))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", err)
	}

	logger.Info("Server exited properly")
}
