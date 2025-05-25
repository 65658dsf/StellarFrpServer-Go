package apis

import (
	"stellarfrp/internal/api/handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes 注册不需要认证的公共API路由
func RegisterPublicRoutes(
	router *gin.RouterGroup,
	userHandler *handler.UserHandler,
	systemHandler *handler.SystemHandler,
	announcementHandler *handler.AnnouncementHandler,
	adHandler *handler.AdHandler,
) {
	// 用户登录注册相关路由
	users := router.Group("/users")
	{
		users.POST("/register", userHandler.Register)
		users.POST("/sendcode", userHandler.SendMessage)
		users.POST("/login", userHandler.Login)
		users.POST("/resetpwd", userHandler.ResetPassword)
	}

	// 系统公告相关路由
	router.GET("/system/status", systemHandler.GetSystemStatus)

	// 公告相关路由
	router.GET("/announcements", announcementHandler.GetAnnouncements)

	// 广告相关路由
	router.GET("/ads", adHandler.GetAds)
}

// RegisterAuthRoutes 注册需要认证的API路由
func RegisterAuthRoutes(
	router *gin.RouterGroup,
	userHandler *handler.UserHandler,
	userCheckinHandler *handler.UserCheckinHandler,
	nodeHandler *handler.NodeHandler,
	proxyHandler *handler.ProxyHandler,
	proxyAuthHandler *handler.ProxyAuthHandler,
) {
	// 用户信息相关路由（需要认证）
	users := router.Group("/users")
	{
		users.GET("/info", userHandler.GetUserInfo)
		users.POST("/resettoken", userHandler.ResetToken)

		// 用户签到相关路由
		users.POST("/checkin", userCheckinHandler.Checkin)
		users.GET("/checkin/status", userCheckinHandler.GetCheckinStatus)
		users.GET("/checkin/logs", userCheckinHandler.GetCheckinLogs)
	}

	// 异步任务路由
	tasks := router.Group("/tasks")
	{
		tasks.POST("/", userHandler.CreateAsync)
		tasks.GET("/:id", userHandler.GetTaskStatus)
	}

	// 注册节点相关路由
	RegisterNodeRoutes(router, nodeHandler)

	// 注册隧道相关路由
	RegisterProxyRoutes(router, proxyHandler, proxyAuthHandler)
}

// 保留原有的RegisterRoutes函数以保持兼容性
func RegisterRoutes(
	router *gin.RouterGroup,
	userHandler *handler.UserHandler,
	userCheckinHandler *handler.UserCheckinHandler,
	nodeHandler *handler.NodeHandler,
	proxyHandler *handler.ProxyHandler,
	proxyAuthHandler *handler.ProxyAuthHandler,
	adHandler *handler.AdHandler,
	announcementHandler *handler.AnnouncementHandler,
	systemHandler *handler.SystemHandler,
) {
	// 注册公共路由
	RegisterPublicRoutes(router, userHandler, systemHandler, announcementHandler, adHandler)

	// 注册需要认证的路由
	RegisterAuthRoutes(router, userHandler, userCheckinHandler, nodeHandler, proxyHandler, proxyAuthHandler)
}

// RegisterAdRoutes 注册广告相关路由
func RegisterAdRoutes(router *gin.RouterGroup, adHandler *handler.AdHandler) {
	router.GET("/ads", adHandler.GetAds)
}

// RegisterSystemRoutes 注册系统相关路由
func RegisterSystemRoutes(router *gin.RouterGroup, systemHandler *handler.SystemHandler) {
	router.GET("/system/status", systemHandler.GetSystemStatus)
}
