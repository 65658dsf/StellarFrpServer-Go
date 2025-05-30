package apis

import (
	"stellarfrp/internal/api/handler"

	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes 注册用户相关路由 (认证后)
// 注意：这个函数应该由 RegisterAuthRoutes 调用，传入的 router 已经是认证过的 group
func RegisterUserRoutes(router *gin.RouterGroup, userHandler *handler.UserHandler, userCheckinHandler *handler.UserCheckinHandler, realNameAuthHandler *handler.RealNameAuthHandler) {
	// 用户相关路由，这里的 router 已经是父级 group (例如 /api/v1/users)
	// 所以不需要再 router.Group("/users")
	router.GET("/info", userHandler.GetUserInfo)
	router.POST("/resettoken", userHandler.ResetToken)

	// 用户签到相关路由
	router.POST("/checkin", userCheckinHandler.Checkin)
	router.GET("/checkin/status", userCheckinHandler.GetCheckinStatus)
	router.GET("/checkin/logs", userCheckinHandler.GetCheckinLogs)

	// 实名认证路由
	router.POST("/realname", realNameAuthHandler.RealNameAuth)
	// router.GET("/realname/ping", realNameAuthHandler.Ping) // 测试路由

	// 异步任务路由 (这些也通常在 /users 或特定用户的子资源下，但当前结构是在 /tasks)
	// 如果 tasks 也是用户相关的认证路由，也可以考虑在这里注册或单独处理
	// tasks := router.Group("/tasks") // 这会使路径变成 /api/v1/users/tasks
	// {
	// 	tasks.POST("/", userHandler.CreateAsync)
	// 	tasks.GET("/:id", userHandler.GetTaskStatus)
	// }
}

// RegisterUserPublicRoutes 注册用户相关的公共路由 (不需要认证)
// 例如：注册、登录、发送验证码、重置密码
// (这个函数目前不存在，但可以考虑创建以更好地组织代码)
// func RegisterUserPublicRoutes(router *gin.RouterGroup, userHandler *handler.UserHandler) {
// 	 publicUsers := router.Group("/users")
// 	 {
// 		 publicUsers.POST("/register", userHandler.Register)
// 		 publicUsers.POST("/sendcode", userHandler.SendMessage)
// 		 publicUsers.POST("/login", userHandler.Login)
// 		 publicUsers.POST("/resetpwd", userHandler.ResetPassword)
// 	 }
// }

// 注意：原有的 users.go 文件中的 RegisterUserRoutes 函数只包含 /users 子组的创建和少量路由。
// 本次修改是假设 RegisterUserRoutes 用于注册在已有的 /users group 下的更多端点。
// 如果原始 RegisterUserRoutes 的意图不同，需再调整。
// 按照当前项目的路由注册方式，原有的 RegisterUserRoutes 似乎并没有包含所有 /users 的认证路由。
// 此处修改的 RegisterUserRoutes 是为了被 RegisterAuthRoutes 调用，并传入一个已经是 /users 的 group。
