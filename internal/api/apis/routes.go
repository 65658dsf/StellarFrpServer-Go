package apis

import (
	"stellarfrp/internal/api/handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有API路由
func RegisterRoutes(v1 *gin.RouterGroup, userHandler *handler.UserHandler, nodeHandler *handler.NodeHandler) {
	// 注册用户相关路由
	RegisterUserRoutes(v1, userHandler)

	// 注册节点相关路由
	RegisterNodeRoutes(v1, nodeHandler)
}