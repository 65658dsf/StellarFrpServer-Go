package admin

import (
	"stellarfrp/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes 注册管理员API路由
func RegisterAdminRoutes(router *gin.RouterGroup, userAdminHandler *UserAdminHandler, announcementAdminHandler *AnnouncementAdminHandler, nodeAdminHandler *NodeAdminHandler, groupAdminHandler *GroupAdminHandler, productAdminHandler *ProductAdminHandler, proxyAdminHandler *ProxyAdminHandler) {
	// 用户管理路由
	users := router.Group("/users")
	{
		users.GET("", userAdminHandler.ListUsers)
		users.GET("/:id", userAdminHandler.GetUser)
		users.POST("", userAdminHandler.CreateUser)
		users.POST("/:id", userAdminHandler.UpdateUser)
		users.DELETE("/:id", userAdminHandler.DeleteUser)
		users.GET("/search", userAdminHandler.SearchUsers)
		users.POST("/:id/reset-token", userAdminHandler.ResetUserToken)
	}

	// 公告管理路由
	announcements := router.Group("/announcements")
	announcements.Use(middleware.AdminAuth(userAdminHandler.userService))
	{
		announcements.POST("/create", announcementAdminHandler.CreateAnnouncement)
		announcements.GET("", announcementAdminHandler.GetAdminAnnouncements)
		announcements.GET("/:id", announcementAdminHandler.GetAdminAnnouncementByID)
		announcements.POST("/update", announcementAdminHandler.UpdateAnnouncement)
		announcements.POST("/delete", announcementAdminHandler.DeleteAnnouncement)
	}

	// 节点管理路由
	nodes := router.Group("/nodes")
	nodes.Use(middleware.AdminAuth(userAdminHandler.userService))
	{
		nodes.GET("", nodeAdminHandler.ListNodes)
		nodes.GET("/:id", nodeAdminHandler.GetNode)
		nodes.POST("/create", nodeAdminHandler.CreateNode)
		nodes.POST("/update", nodeAdminHandler.UpdateNode)
		nodes.POST("/delete", nodeAdminHandler.DeleteNode)
		nodes.GET("/search", nodeAdminHandler.SearchNodes)
		// 捐赠节点相关路由
		nodes.GET("/donated", nodeAdminHandler.ListDonatedNodes)
		nodes.POST("/review", nodeAdminHandler.ReviewDonatedNode)
	}

	// 用户组管理路由
	groups := router.Group("/groups")
	groups.Use(middleware.AdminAuth(userAdminHandler.userService))
	{
		groups.GET("", groupAdminHandler.ListGroups)
		groups.GET("/:id", groupAdminHandler.GetGroup)
		groups.POST("/create", groupAdminHandler.CreateGroup)
		groups.POST("/update", groupAdminHandler.UpdateGroup)
		groups.DELETE("/:id", groupAdminHandler.DeleteGroup)
		groups.GET("/search", groupAdminHandler.SearchGroups)
	}

	// 商品管理路由
	products := router.Group("/products")
	products.Use(middleware.AdminAuth(userAdminHandler.userService))
	{
		products.GET("", productAdminHandler.ListProducts)
		products.GET("/:id", productAdminHandler.GetProduct)
		products.POST("/create", productAdminHandler.CreateProduct)
		products.POST("/update", productAdminHandler.UpdateProduct)
		products.POST("/delete", productAdminHandler.DeleteProduct)
	}

	// 订单管理路由
	orders := router.Group("/orders")
	orders.Use(middleware.AdminAuth(userAdminHandler.userService))
	{
		orders.GET("", productAdminHandler.ListOrders)
		orders.GET("/:order_no", productAdminHandler.GetOrder)
		orders.POST("/update", productAdminHandler.UpdateOrderStatus)
		orders.POST("/delete", productAdminHandler.DeleteOrder)
	}

	// 隧道管理路由
	proxies := router.Group("/proxies")
	proxies.Use(middleware.AdminAuth(userAdminHandler.userService))
	{
		proxies.GET("", proxyAdminHandler.ListProxies)
		proxies.GET("/status", proxyAdminHandler.ListAllProxiesStatus)
		proxies.GET("/:id/status", proxyAdminHandler.GetProxyStatus)
		proxies.GET("/search", proxyAdminHandler.SearchProxies)
		proxies.POST("/close", proxyAdminHandler.CloseProxy)
		proxies.POST("/user/close", proxyAdminHandler.CloseUserProxies)
	}
}
