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
		// 获取节点信息（在线状态、流量等）
		nodes.GET("/info", nodeHandler.GetNodesInfo)
		// 捐赠节点
		nodes.POST("/donate", nodeHandler.DonateNode)
		// 获取用户自己的节点
		nodes.GET("/my", nodeHandler.GetUserNodes)
		// 接收节点负载信息
		nodes.POST("/load/webhook", nodeHandler.ReceiveNodeLoad)
	}
}
