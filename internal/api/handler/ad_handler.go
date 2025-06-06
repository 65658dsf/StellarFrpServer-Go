package handler

import (
	"net/http"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"

	"github.com/gin-gonic/gin"
)

// AdHandler 广告处理器
type AdHandler struct {
	adService *service.AdService
	logger    *logger.Logger
}

// NewAdHandler 创建广告处理器实例
func NewAdHandler(adService *service.AdService, logger *logger.Logger) *AdHandler {
	return &AdHandler{
		adService: adService,
		logger:    logger,
	}
}

// GetAds 获取广告列表
// @Summary 获取广告列表
// @Description 获取当前活跃的广告列表
// @Tags 广告
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功"
// @Router /api/v1/ads [get]
func (h *AdHandler) GetAds(c *gin.Context) {
	ctx := c.Request.Context()
	ads, err := h.adService.GetActiveAds(ctx)
	if err != nil {
		h.logger.Error("获取广告列表失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "获取广告列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": ads,
	})
}
