package apis

import (
	"stellarfrp/internal/api/handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有API路由
func RegisterRoutes(v1 *gin.RouterGroup, userHandler *handler.UserHandler, userCheckinHandler *handler.UserCheckinHandler, nodeHandler *handler.NodeHandler, proxyHandler *handler.ProxyHandler, proxyAuthHandler *handler.ProxyAuthHandler) {
	// 注册用户相关路由
	RegisterUserRoutes(v1, userHandler, userCheckinHandler)

	// 注册节点相关路由
	RegisterNodeRoutes(v1, nodeHandler)

	// 注册隧道相关路由
	RegisterProxyRoutes(v1, proxyHandler, proxyAuthHandler)
}
