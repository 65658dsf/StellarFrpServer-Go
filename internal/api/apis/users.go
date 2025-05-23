package apis

import (
	"stellarfrp/internal/api/handler"

	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes 注册用户相关路由
func RegisterUserRoutes(router *gin.RouterGroup, userHandler *handler.UserHandler, userCheckinHandler *handler.UserCheckinHandler) {
	// 用户相关路由
	users := router.Group("/users")
	{
		users.POST("/register", userHandler.Register)
		users.POST("/sendcode", userHandler.SendMessage)
		users.POST("/login", userHandler.Login)
		users.POST("/resetpwd", userHandler.ResetPassword)
		users.GET("/info", userHandler.GetUserInfo)
		users.POST("/resettoken", userHandler.ResetToken)

		// 用户签到相关路由
		users.POST("/checkin", userCheckinHandler.Checkin)
		users.GET("/checkin/status", userCheckinHandler.GetCheckinStatus)
		users.GET("/checkin/logs", userCheckinHandler.GetCheckinLogs)
	}

	// 异步任务路由
	tasks := router.Group("/tasks")
	{
		tasks.POST("/", userHandler.CreateAsync)
		tasks.GET("/:id", userHandler.GetTaskStatus)
	}
}
