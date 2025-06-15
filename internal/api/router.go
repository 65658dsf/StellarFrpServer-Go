package api

import (
	"stellarfrp/config"
	"stellarfrp/internal/api/admin"
	"stellarfrp/internal/api/apis"
	"stellarfrp/internal/api/handler"
	"stellarfrp/internal/middleware"
	"stellarfrp/internal/repository"
	"stellarfrp/internal/scheduler"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/async"
	"stellarfrp/pkg/email"
	"stellarfrp/pkg/geetest"
	"stellarfrp/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// SetupRouter 设置API路由
func SetupRouter(cfg *config.Config, logger *logger.Logger, db *sqlx.DB, redisClient *redis.Client, geetestClient *geetest.GeetestClient, realNameAuthHandler *handler.RealNameAuthHandler) *gin.Engine {
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
	nodeRepo := repository.NewNodeRepository(db)
	nodeTrafficRepo := repository.NewNodeTrafficRepository(db)
	proxyRepo := repository.NewProxyRepository(db)
	userCheckinRepo := repository.NewUserCheckinRepository(db)
	userTrafficLogRepo := repository.NewUserTrafficLogRepository(db)
	adRepo := repository.NewAdRepository(db)
	announcementRepo := repository.NewAnnouncementRepository(db)
	systemRepo := repository.NewSystemRepository(db)
	productRepo := repository.NewProductRepository(db)
	orderRepo := repository.NewOrderRepository(db)

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
	userService := service.NewUserService(userRepo, groupRepo, userTrafficLogRepo, redisClient, worker, emailService, logger)
	nodeService := service.NewNodeService(nodeRepo, nodeTrafficRepo)
	proxyService := service.NewProxyService(proxyRepo, nodeService, userService, redisClient, logger)
	nodeTrafficService := service.NewNodeTrafficService(nodeRepo, nodeTrafficRepo, logger)
	userCheckinService := service.NewUserCheckinService(userRepo, groupRepo, userCheckinRepo, redisClient, logger)
	userTrafficLogService := service.NewUserTrafficLogService(nodeRepo, userTrafficLogRepo, redisClient, logger)
	adService := service.NewAdService(adRepo, redisClient, logger)
	announcementService := service.NewAnnouncementService(announcementRepo, redisClient, logger)
	systemService := service.NewSystemService(systemRepo, redisClient, logger)
	groupService := service.NewGroupService(groupRepo, logger)
	productService := service.NewProductService(productRepo, orderRepo, userService, redisClient, logger)

	// 初始化节点调度器
	nodeScheduler := scheduler.NewNodeScheduler(nodeTrafficService, logger)
	nodeScheduler.Start() // 启动节点调度

	// 初始化流量记录调度器
	trafficScheduler := scheduler.NewTrafficScheduler(userTrafficLogService, userService, proxyService, nodeService, logger)
	trafficScheduler.Start() // 启动流量记录调度

	// 初始化处理器
	userHandler := handler.NewUserHandler(userService, redisClient, emailService, logger, geetestClient, proxyService)
	userCheckinHandler := handler.NewUserCheckinHandler(userService, userCheckinService, logger)
	nodeHandler := handler.NewNodeHandler(nodeService, userService, logger, redisClient)
	proxyHandler := handler.NewProxyHandler(proxyService, nodeService, userService, logger)
	proxyAuthHandler := handler.NewProxyAuthHandler(proxyService, userService, userTrafficLogService, logger)
	adHandler := handler.NewAdHandler(adService, logger)
	announcementHandler := handler.NewAnnouncementHandler(announcementService, logger)
	systemHandler := handler.NewSystemHandler(systemService, logger)
	productHandler := handler.NewProductHandler(productService, logger)

	// 初始化管理员处理器
	userAdminHandler := admin.NewUserAdminHandler(userService, logger)
	announcementAdminHandler := admin.NewAnnouncementAdminHandler(announcementService, logger)
	nodeAdminHandler := admin.NewNodeAdminHandler(nodeService, nodeRepo, userService, logger)
	groupAdminHandler := admin.NewGroupAdminHandler(groupService, logger)
	productAdminHandler := admin.NewProductAdminHandler(productService, userService, logger)
	proxyAdminHandler := admin.NewProxyAdminHandler(proxyService, nodeService, userService, redisClient, logger)

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API版本v1
	v1 := router.Group("/api/v1")

	// 创建需要认证的API路由组
	authRouter := v1.Group("")
	// 为需要认证的API路由添加UserAuth中间件
	authRouter.Use(middleware.UserAuth(userService))

	// 注册不需要认证的路由（如登录、注册、发送验证码等）
	apis.RegisterPublicRoutes(v1, userHandler, systemHandler, announcementHandler, adHandler, proxyAuthHandler, productHandler)

	// 注册需要认证的API路由
	apis.RegisterAuthRoutes(authRouter, userHandler, userCheckinHandler, nodeHandler, proxyHandler, proxyAuthHandler, realNameAuthHandler, productHandler)

	// 注册管理员API路由
	adminRouter := v1.Group("/admin")
	// 添加管理员认证中间件
	adminRouter.Use(middleware.AdminAuth(userService))
	admin.RegisterAdminRoutes(adminRouter, userAdminHandler, announcementAdminHandler, nodeAdminHandler, groupAdminHandler, productAdminHandler, proxyAdminHandler)

	return router
}
