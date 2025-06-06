package apis

import (
	"stellarfrp/internal/api/handler"

	"github.com/gin-gonic/gin"
)

// RegisterShopRoutes 注册商品相关路由（需要认证的部分）
func RegisterShopRoutes(router *gin.RouterGroup, productHandler *handler.ProductHandler) {
	shop := router.Group("/shop")
	{
		// 需要认证的路由
		shop.GET("/orders", productHandler.GetUserOrders)
		shop.GET("/order/status", productHandler.GetOrderStatus)
		shop.POST("/order/create", productHandler.CreateOrderLink)
	}
}

// RegisterShopPublicRoutes 注册商品相关公开路由
func RegisterShopPublicRoutes(router *gin.RouterGroup, productHandler *handler.ProductHandler) {
	// 获取商品列表（公开路由）
	router.GET("/shop/products", productHandler.GetProducts)

	// 爱发电Webhook回调路由（公开路由）
	router.POST("/afdian/webhook/stellarFrpWEbHOOk", productHandler.AfdianWebhook)
}
