package middleware

import (
	"stellarfrp/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 日志中间件
func Logger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算耗时
		timeStamp := time.Now()
		latency := timeStamp.Sub(start)

		if raw != "" {
			path = path + "?" + raw
		}

		log.Info("访问日志",
			"状态码", c.Writer.Status(),
			"延迟", latency,
			"客户端IP", c.ClientIP(),
			"方法", c.Request.Method,
			"路径", path,
		)
	}
}

// Recovery 恢复中间件
func Recovery(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("服务器错误", "error", err)
				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}