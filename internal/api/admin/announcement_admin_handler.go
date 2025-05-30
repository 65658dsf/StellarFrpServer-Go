package admin

import (
	"net/http"
	"stellarfrp/internal/model"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// AnnouncementAdminHandler 公告管理处理器
type AnnouncementAdminHandler struct {
	announcementService *service.AnnouncementService
	logger              *logger.Logger
}

// NewAnnouncementAdminHandler 创建公告管理处理器实例
func NewAnnouncementAdminHandler(announcementService *service.AnnouncementService, logger *logger.Logger) *AnnouncementAdminHandler {
	return &AnnouncementAdminHandler{
		announcementService: announcementService,
		logger:              logger,
	}
}

// CreateAnnouncementRequest 创建公告请求结构体
type CreateAnnouncementRequest struct {
	Title       string `json:"title" binding:"required"`
	Content     string `json:"content" binding:"required"`
	IsImportant bool   `json:"is_important"`
	IsVisible   bool   `json:"is_visible"`
	Author      string `json:"author"`
}

// CreateAnnouncement 创建公告
// @Summary 创建公告
// @Description 管理员创建新公告
// @Tags 公告管理
// @Accept json
// @Produce json
// @Param announcement body CreateAnnouncementRequest true "公告信息"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /admin/announcements/create [post]
func (h *AnnouncementAdminHandler) CreateAnnouncement(c *gin.Context) {
	var req CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("创建公告参数绑定失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "参数错误：" + err.Error(),
		})
		return
	}

	author := req.Author
	if author == "" {
		author = "系统管理员"
	}

	announcement := &model.Announcement{
		Title:       req.Title,
		Content:     req.Content,
		IsImportant: req.IsImportant,
		PublishDate: time.Now(),
		Author:      author,
		IsVisible:   req.IsVisible,
	}

	ctx := c.Request.Context()
	if err := h.announcementService.CreateAnnouncement(ctx, announcement); err != nil {
		h.logger.Error("创建公告失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "创建公告失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": announcement,
	})
}

// UpdateAnnouncementRequest 更新公告请求结构体
type UpdateAnnouncementRequest struct {
	ID          int64   `json:"id" binding:"required"`
	Title       *string `json:"title"`
	Content     *string `json:"content"`
	IsImportant *bool   `json:"is_important"`
	IsVisible   *bool   `json:"is_visible"`
	Author      *string `json:"author"`
}

// UpdateAnnouncement 更新公告
// @Summary 更新公告
// @Description 管理员更新现有公告
// @Tags 公告管理
// @Accept json
// @Produce json
// @Param announcement body UpdateAnnouncementRequest true "公告ID和公告信息"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /admin/announcements/update [post]
func (h *AnnouncementAdminHandler) UpdateAnnouncement(c *gin.Context) {
	var req UpdateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("更新公告参数绑定失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "参数错误：" + err.Error(),
		})
		return
	}

	id := req.ID // 从请求体获取ID

	ctx := c.Request.Context()
	announcement, err := h.announcementService.GetAnnouncementByID(ctx, id) // 假设service层有GetAnnouncementByID方法
	if err != nil {
		h.logger.Error("获取待更新公告失败", "id", id, "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 404,
			"msg":  "公告不存在或获取失败",
		})
		return
	}

	if req.Title != nil {
		announcement.Title = *req.Title
	}
	if req.Content != nil {
		announcement.Content = *req.Content
	}
	if req.IsImportant != nil {
		announcement.IsImportant = *req.IsImportant
	}
	if req.IsVisible != nil {
		announcement.IsVisible = *req.IsVisible
	}
	if req.Author != nil {
		announcement.Author = *req.Author
	}

	if err := h.announcementService.UpdateAnnouncement(ctx, announcement); err != nil {
		h.logger.Error("更新公告失败", "id", id, "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "更新公告失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": announcement,
	})
}

// DeleteAnnouncementRequest 删除公告请求结构体
type DeleteAnnouncementRequest struct {
	ID int64 `json:"id" binding:"required"`
}

// DeleteAnnouncement 删除公告
// @Summary 删除公告
// @Description 管理员删除公告
// @Tags 公告管理
// @Accept json
// @Produce json
// @Param announcement body DeleteAnnouncementRequest true "公告ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /admin/announcements/delete [post]
func (h *AnnouncementAdminHandler) DeleteAnnouncement(c *gin.Context) {
	var req DeleteAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("删除公告参数绑定失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "参数错误：" + err.Error(),
		})
		return
	}

	id := req.ID // 从请求体获取ID

	ctx := c.Request.Context()
	if err := h.announcementService.DeleteAnnouncement(ctx, id); err != nil {
		h.logger.Error("删除公告失败", "id", id, "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "删除公告失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
	})
}

// GetAdminAnnouncements 获取公告列表（管理员）
// @Summary 获取公告列表（管理员）
// @Description 管理员获取公告列表，支持分页，包含不可见公告
// @Tags 公告管理
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页条数，默认10"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /admin/announcements [get]
func (h *AnnouncementAdminHandler) GetAdminAnnouncements(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	ctx := c.Request.Context()
	paginatedAnnouncements, err := h.announcementService.GetAnnouncementsAdmin(ctx, page, pageSize)
	if err != nil {
		h.logger.Error("获取管理员公告列表失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "获取公告列表失败",
		})
		return
	}

	totalPages := int64(0)
	if paginatedAnnouncements.Total > 0 && pageSize > 0 {
		totalPages = (paginatedAnnouncements.Total + int64(pageSize) - 1) / int64(pageSize)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取成功",
		"pagination": gin.H{
			"page":      page,
			"page_size": pageSize,
			"pages":     totalPages,
			"total":     paginatedAnnouncements.Total,
		},
		"announcements": paginatedAnnouncements.Items,
	})
}

// GetAdminAnnouncementByID 获取单个公告详情（管理员）
// @Summary 获取单个公告详情（管理员）
// @Description 管理员根据ID获取公告详情，包含不可见公告
// @Tags 公告管理
// @Accept json
// @Produce json
// @Param id path int true "公告ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /admin/announcements/{id} [get]
func (h *AnnouncementAdminHandler) GetAdminAnnouncementByID(c *gin.Context) {
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
	// 此处调用 GetAnnouncementByIDAdmin，假设 service 层有此方法获取单个公告（包括不可见）
	announcement, err := h.announcementService.GetAnnouncementByIDAdmin(ctx, id)
	if err != nil {
		h.logger.Error("获取管理员公告详情失败", "id", id, "error", err)
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
