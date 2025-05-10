package apis

import (
	"stellarfrp/internal/api/handler"

	"github.com/gin-gonic/gin"
)

// RegisterAnnouncementRoutes 注册公告相关路由
func RegisterAnnouncementRoutes(router *gin.RouterGroup, announcementHandler *handler.AnnouncementHandler) {
	// 公告相关路由
	router.GET("/announcements", announcementHandler.GetAnnouncements)
	router.GET("/announcements/:id", announcementHandler.GetAnnouncementByID)
}
