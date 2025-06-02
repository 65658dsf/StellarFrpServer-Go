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
	proxyAuthHandler *handler.ProxyAuthHandler,
) {
	// 用户登录注册相关路由
	users := router.Group("/users")
	{
		users.POST("/register", userHandler.Register)
		users.POST("/sendcode", userHandler.SendMessage)
		users.POST("/login", userHandler.Login)
		users.POST("/resetpwd", userHandler.ResetPassword)
		users.GET("/groups", userHandler.GetGroupList)
	}

	// 系统公告相关路由
	router.GET("/system/status", systemHandler.GetSystemStatus)

	// 公告相关路由
	router.GET("/announcements", announcementHandler.GetAnnouncements)

	// 广告相关路由
	router.GET("/ads", adHandler.GetAds)

	// 隧道鉴权路由（不需要认证）
	router.POST("/proxy/auth", proxyAuthHandler.HandleProxyAuth)
}

// RegisterAuthRoutes 注册需要认证的API路由
func RegisterAuthRoutes(
	router *gin.RouterGroup,
	userHandler *handler.UserHandler,
	userCheckinHandler *handler.UserCheckinHandler,
	nodeHandler *handler.NodeHandler,
	proxyHandler *handler.ProxyHandler,
	proxyAuthHandler *handler.ProxyAuthHandler,
	realNameAuthHandler *handler.RealNameAuthHandler,
) {
	// 用户信息、签到、实名认证等路由 (需要认证)
	usersGroup := router.Group("/users")                                                 // 创建 /users 子分组
	RegisterUserRoutes(usersGroup, userHandler, userCheckinHandler, realNameAuthHandler) // 调用users.go中的函数

	// 异步任务路由 (从原来的 users.go 中移到这里，或者保持独立)
	// 如果 tasks 路由也是认证路由，且希望在 /api/v1/tasks 下
	tasksGroup := router.Group("/tasks")
	{
		tasksGroup.POST("/", userHandler.CreateAsync) // 注意 userHandler 是否合适，或者需要 TaskHandler
		tasksGroup.GET("/:id", userHandler.GetTaskStatus)
	}

	// 注册节点相关路由
	RegisterNodeRoutes(router, nodeHandler) // router 是 /api/v1 (认证过的)

	// 注册隧道相关路由
	RegisterProxyRoutes(router, proxyHandler, proxyAuthHandler) // router 是 /api/v1 (认证过的)
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
	realNameAuthHandler *handler.RealNameAuthHandler,
) {
	// 注册公共路由
	RegisterPublicRoutes(router, userHandler, systemHandler, announcementHandler, adHandler, proxyAuthHandler)

	// 注册需要认证的路由
	RegisterAuthRoutes(router, userHandler, userCheckinHandler, nodeHandler, proxyHandler, proxyAuthHandler, realNameAuthHandler)
}

// RegisterAdRoutes 注册广告相关路由
func RegisterAdRoutes(router *gin.RouterGroup, adHandler *handler.AdHandler) {
	router.GET("/ads", adHandler.GetAds)
}

// RegisterSystemRoutes 注册系统相关路由
func RegisterSystemRoutes(router *gin.RouterGroup, systemHandler *handler.SystemHandler) {
	router.GET("/system/status", systemHandler.GetSystemStatus)
}
