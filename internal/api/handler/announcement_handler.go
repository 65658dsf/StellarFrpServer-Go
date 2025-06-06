package handler

import (
	"net/http"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AnnouncementHandler 公告处理器
type AnnouncementHandler struct {
	announcementService *service.AnnouncementService
	logger              *logger.Logger
}

// NewAnnouncementHandler 创建公告处理器实例
func NewAnnouncementHandler(announcementService *service.AnnouncementService, logger *logger.Logger) *AnnouncementHandler {
	return &AnnouncementHandler{
		announcementService: announcementService,
		logger:              logger,
	}
}

// GetAnnouncements 获取公告列表
// @Summary 获取公告列表
// @Description 获取公告列表，支持分页
// @Tags 公告
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param limit query int false "每页条数，默认10"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /api/v1/announcements [get]
func (h *AnnouncementHandler) GetAnnouncements(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}

	ctx := c.Request.Context()
	paginatedAnnouncements, err := h.announcementService.GetAnnouncements(ctx, page, limit)
	if err != nil {
		h.logger.Error("获取公告列表失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "获取公告列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": paginatedAnnouncements,
	})
}

// GetAnnouncementByID 获取公告详情
// @Summary 获取公告详情
// @Description 根据ID获取公告详情
// @Tags 公告
// @Accept json
// @Produce json
// @Param id path int true "公告ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /api/v1/announcements/{id} [get]
func (h *AnnouncementHandler) GetAnnouncementByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "无效的公告ID",
		})
		return
	}

	ctx := c.Request.Context()
	announcement, err := h.announcementService.GetAnnouncementByID(ctx, id)
	if err != nil {
		h.logger.Error("获取公告详情失败", "id", id, "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 404,
			"msg":  "公告不存在或获取失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": announcement,
	})
}
