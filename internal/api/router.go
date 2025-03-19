package api

import (
	"stellarfrp/config"
	"stellarfrp/internal/api/apis"
	"stellarfrp/internal/api/handler"
	"stellarfrp/internal/middleware"
	"stellarfrp/internal/repository"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/async"
	"stellarfrp/pkg/email"
	"stellarfrp/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// SetupRouter 设置API路由
func SetupRouter(cfg *config.Config, logger *logger.Logger, db *sqlx.DB, redisClient *redis.Client) *gin.Engine {
	// 创建Gin引擎
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	// 使用中间件
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.CORS())

	// 创建异步工作器
	worker := async.NewWorker(100, logger)
	worker.Start(5) // 启动5个工作协程
	defer worker.Stop()

	// 初始化存储库
	userRepo := repository.NewUserRepository(db)
	groupRepo := repository.NewGroupRepository(db)

	// 初始化邮件服务
	emailService := email.NewService(email.Config{
		Host:     cfg.Email.Host,
		Port:     cfg.Email.Port,
		Username: cfg.Email.Username,
		Password: cfg.Email.Password,
		From:     cfg.Email.From,
		FromName: cfg.Email.FromName,
	}, logger)

	// 初始化服务
	userService := service.NewUserService(userRepo, groupRepo, redisClient, worker, emailService, logger)

	// 初始化处理器
	userHandler := handler.NewUserHandler(userService, redisClient, emailService, logger)

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API版本v1
	v1 := router.Group("/api/v1")

	// 注册所有API路由
	apis.RegisterRoutes(v1, userHandler)

	return router
}
