package apis

import (
	"stellarfrp/internal/api/handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有API路由
func RegisterRoutes(v1 *gin.RouterGroup, userHandler *handler.UserHandler) {
	// 注册用户相关路由
	RegisterUserRoutes(v1, userHandler)

	// 这里可以添加其他模块的路由注册
	// 例如：RegisterProductRoutes(v1, productHandler)
}