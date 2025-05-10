package apis

import (
	"stellarfrp/internal/api/handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有API路由
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
	// 注册用户相关路由
	RegisterUserRoutes(router, userHandler, userCheckinHandler)

	// 注册节点相关路由
	RegisterNodeRoutes(router, nodeHandler)

	// 注册隧道相关路由
	RegisterProxyRoutes(router, proxyHandler, proxyAuthHandler)

	// 注册广告相关路由
	RegisterAdRoutes(router, adHandler)

	// 注册公告相关路由
	RegisterAnnouncementRoutes(router, announcementHandler)

	// 注册系统相关路由
	RegisterSystemRoutes(router, systemHandler)
}

// RegisterAdRoutes 注册广告相关路由
func RegisterAdRoutes(router *gin.RouterGroup, adHandler *handler.AdHandler) {
	router.GET("/ads", adHandler.GetAds)
}

// RegisterSystemRoutes 注册系统相关路由
func RegisterSystemRoutes(router *gin.RouterGroup, systemHandler *handler.SystemHandler) {
	router.GET("/system/status", systemHandler.GetSystemStatus)
}
