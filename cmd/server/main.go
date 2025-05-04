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
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 初始化日志
	logger := logger.NewLoggerWithConfig(cfg.LogLevel, cfg.LogFile)
	defer logger.Sync()

	// 初始化数据库连接
	db, err := database.NewMySQLConnection(cfg.Database)
	if err != nil {
		logger.Fatal("无法链接到数据库", err)
	}
	defer db.Close()

	// 初始化Redis连接
	redisClient, err := database.NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Fatal("无法链接到Redis", err)
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
		logger.Info(fmt.Sprintf("服务器启动于端口: %d", cfg.APIPort))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("启动服务器失败", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("服务器被强制关闭", err)
	}

	logger.Info("服务器已正常退出")
}
