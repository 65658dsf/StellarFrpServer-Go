package handler

import (
	"net/http"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"

	"github.com/gin-gonic/gin"
)

// SystemHandler 系统状态处理器
type SystemHandler struct {
	systemService *service.SystemService
	logger        *logger.Logger
}

// NewSystemHandler 创建系统状态处理器实例
func NewSystemHandler(systemService *service.SystemService, logger *logger.Logger) *SystemHandler {
	return &SystemHandler{
		systemService: systemService,
		logger:        logger,
	}
}

// GetSystemStatus 获取系统状态
// @Summary 获取系统状态
// @Description 获取系统统计信息，包括用户总数、隧道总数、总流量和节点总数
// @Tags 系统
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功"
// @Router /api/system/status [get]
func (h *SystemHandler) GetSystemStatus(c *gin.Context) {
	status, err := h.systemService.GetSystemStatus(c.Request.Context())
	if err != nil {
		h.logger.Error("获取系统状态失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "获取系统状态失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取成功",
		"data": status,
	})
}
