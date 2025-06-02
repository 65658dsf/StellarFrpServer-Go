package middleware

import (
	"context"
	"net/http"
	"stellarfrp/internal/constants"
	"stellarfrp/internal/service"

	"github.com/gin-gonic/gin"
)

// UserAuth 用户认证中间件
func UserAuth(userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取Token
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusOK, gin.H{"code": 401, "msg": constants.ErrUnauthorized})
			c.Abort()
			return
		}

		// 验证Token
		user, err := userService.GetByToken(context.Background(), token)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 401, "msg": constants.ErrInvalidToken})
			c.Abort()
			return
		}

		// 检查用户是否在黑名单中
		isBlacklisted, err := userService.IsUserBlacklistedByToken(context.Background(), token)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": constants.ErrInternalServer})
			c.Abort()
			return
		}

		if isBlacklisted {
			c.JSON(http.StatusOK, gin.H{"code": 403, "msg": constants.ErrBlacklisted})
			c.Abort()
			return
		}

		// 将用户ID存储到上下文中
		c.Set("user_id", user.ID)

		// 处理用户组ID逻辑：
		// 如果用户未实名认证(is_verified=0)且不是黑名单用户(group_id!=6)，则视为未实名用户组(group_id=1)
		effectiveGroupID := user.GroupID
		if user.IsVerified == 0 && user.GroupID != 6 {
			effectiveGroupID = 1 // 未实名用户组ID为1
		}

		// 将实际使用的GroupID存储到上下文中，供后续处理使用
		c.Set("group_id", effectiveGroupID)
		c.Next()
	}
}
