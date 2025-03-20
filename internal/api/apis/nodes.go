package apis

import (
	"stellarfrp/internal/api/handler"

	"github.com/gin-gonic/gin"
)

// RegisterNodeRoutes 注册节点相关路由
func RegisterNodeRoutes(router *gin.RouterGroup, nodeHandler *handler.NodeHandler) {
	// 节点相关路由
	nodes := router.Group("/nodes")
	{
		// 获取用户可访问的节点列表
		nodes.GET("/get", nodeHandler.GetAccessibleNodes)
	}
}
