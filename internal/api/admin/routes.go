package admin

import (
	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes 注册管理员API路由
func RegisterAdminRoutes(router *gin.RouterGroup, userAdminHandler *UserAdminHandler) {
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
}
