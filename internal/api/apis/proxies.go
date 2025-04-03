package apis

import (
	"stellarfrp/internal/api/handler"

	"github.com/gin-gonic/gin"
)

// RegisterProxyRoutes 注册隧道相关路由
func RegisterProxyRoutes(router *gin.RouterGroup, proxyHandler *handler.ProxyHandler) {
	// 隧道相关路由
	proxies := router.Group("/proxies")
	{
		// 创建隧道
		proxies.POST("/create", proxyHandler.CreateProxy)
		// 获取用户的隧道列表
		proxies.GET("/list", proxyHandler.GetProxies)
		// 更新隧道
		proxies.PUT("/update", proxyHandler.UpdateProxy)
		// 删除隧道
		proxies.POST("/delete", proxyHandler.DeleteProxy)
		// 根据ID获取隧道
		proxies.GET("/get/:id", proxyHandler.GetProxyByID)
	}
}
