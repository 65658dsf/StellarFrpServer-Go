package apis

import (
	"stellarfrp/internal/api/handler"

	"github.com/gin-gonic/gin"
)

// RegisterProxyRoutes 注册隧道相关路由
func RegisterProxyRoutes(router *gin.RouterGroup, proxyHandler *handler.ProxyHandler, proxyAuthHandler *handler.ProxyAuthHandler) {
	// 隧道相关路由
	proxies := router.Group("/proxy")
	{
		// 创建隧道
		proxies.POST("/create", proxyHandler.CreateProxy)
		// 更新隧道
		proxies.POST("/edit", proxyHandler.UpdateProxy)
		// 删除隧道
		proxies.POST("/delete", proxyHandler.DeleteProxy)
		// 获取隧道
		proxies.GET("/get", proxyHandler.GetProxyByID)
		// 获取隧道状态
		proxies.GET("/status", proxyHandler.GetProxyStatus)
		proxies.POST("/status", proxyHandler.GetProxyStatus)
		// 关闭隧道
		proxies.POST("/close", proxyHandler.CloseProxy)
		// 注册FRP隧道鉴权路由已移至公共路由，这里不再注册
		// proxies.POST("/auth", proxyAuthHandler.HandleProxyAuth)
	}
}
