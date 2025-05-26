package middleware

import (
	"context"
	"net/http"
	"stellarfrp/internal/constants"
	"stellarfrp/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminAuth 管理员认证中间件
func AdminAuth(userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取Token
		token := c.GetHeader("Authorization")
		// 验证Token
		user, err := userService.GetByToken(context.Background(), token)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 401, "msg": constants.ErrInvalidToken})
			c.Abort()
			return
		}

		// 检查用户是否为管理员（假设管理员的GroupID为4）
		if user.GroupID != 4 {
			c.JSON(http.StatusOK, gin.H{"code": 403, "msg": constants.ErrInsufficientPermission})
			c.Abort()
			return
		}

		// 将用户ID存储到上下文中
		c.Set("user_id", user.ID)
		c.Next()
	}
}
