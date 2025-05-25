package middleware

import (
	"context"
	"net/http"
	"stellarfrp/internal/service"

	"github.com/gin-gonic/gin"
)

// UserAuth 用户认证中间件
func UserAuth(userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取Token
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未授权，请先登录"})
			c.Abort()
			return
		}

		// 验证Token
		user, err := userService.GetByToken(context.Background(), token)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的Token"})
			c.Abort()
			return
		}

		// 检查用户是否在黑名单组（ID为6）
		if user.GroupID == 6 {
			c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "您的账号已被列入黑名单，禁止访问API"})
			c.Abort()
			return
		}

		// 将用户ID和GroupID存储到上下文中，供后续处理使用
		c.Set("user_id", user.ID)
		c.Set("group_id", user.GroupID)
		c.Next()
	}
}
