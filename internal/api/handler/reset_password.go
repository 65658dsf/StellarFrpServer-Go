package handler

import (
	"context"
	"net/http"
	"regexp"
	"stellarfrp/internal/types"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// ResetPassword 重置密码
func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req types.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证邮箱格式
	if !regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@(qq\.com|163\.com|outlook\.com|gmail\.com)$`).MatchString(req.Email) {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "邮箱格式不正确"})
		return
	}

	// 验证密码格式
	if !regexp.MustCompile(`^[A-Za-z0-9]{6,}$`).MatchString(req.Password) || !regexp.MustCompile(`[A-Za-z]`).MatchString(req.Password) || !regexp.MustCompile(`[0-9]`).MatchString(req.Password) {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "密码必须为英文加数字长度在6位以上的密码"})
		return
	}

	// 验证邮箱验证码
	codeKey := "email_verify:" + req.Email
	code, err := h.redisClient.Get(context.Background(), codeKey).Result()
	if err == redis.Nil || code != req.Code {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "验证码错误或已过期"})
		return
	}

	// 使用分布式锁控制并发
	lockKey := "reset_password:" + req.Email
	lock := h.redisClient.SetNX(context.Background(), lockKey, "1", 10*time.Second)
	if !lock.Val() {
		c.JSON(http.StatusOK, gin.H{"code": 429, "msg": "请求过于频繁，请稍后重试"})
		return
	}
	defer h.redisClient.Del(context.Background(), lockKey)

	// 获取用户信息
	user, err := h.userService.GetByEmail(context.Background(), req.Email)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "用户不存在"})
		return
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	// 更新用户密码
	user.Password = string(hashedPassword)
	if err := h.userService.Update(context.Background(), user); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "重置密码失败"})
		return
	}

	// 删除验证码
	h.redisClient.Del(context.Background(), codeKey)

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "密码重置成功"})
}
