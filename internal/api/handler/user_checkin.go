package handler

import (
	"context"
	"net/http"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UserCheckinHandler 用户签到处理器
type UserCheckinHandler struct {
	userService        service.UserService
	userCheckinService service.UserCheckinService
	logger             *logger.Logger
}

// NewUserCheckinHandler 创建用户签到处理器实例
func NewUserCheckinHandler(
	userService service.UserService,
	userCheckinService service.UserCheckinService,
	logger *logger.Logger,
) *UserCheckinHandler {
	return &UserCheckinHandler{
		userService:        userService,
		userCheckinService: userCheckinService,
		logger:             logger,
	}
}

// 格式化流量大小
func formatCheckinTraffic(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return strconv.FormatFloat(float64(bytes)/TB, 'f', 2, 64) + " TB"
	case bytes >= GB:
		return strconv.FormatFloat(float64(bytes)/GB, 'f', 2, 64) + " GB"
	case bytes >= MB:
		return strconv.FormatFloat(float64(bytes)/MB, 'f', 2, 64) + " MB"
	case bytes >= KB:
		return strconv.FormatFloat(float64(bytes)/KB, 'f', 2, 64) + " KB"
	default:
		return strconv.FormatInt(bytes, 10) + " B"
	}
}

// Checkin 用户签到
func (h *UserCheckinHandler) Checkin(c *gin.Context) {
	// 从Header中获取token并验证
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "请先登录"})
		return
	}

	// 验证token并获取用户信息
	user, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的认证信息"})
		return
	}

	// 检查用户状态
	if user.Status != 1 {
		c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "账户已被禁用"})
		return
	}

	// 执行签到
	checkinLog, err := h.userCheckinService.Checkin(context.Background(), user.ID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	// 格式化流量显示
	formattedTraffic := formatCheckinTraffic(checkinLog.RewardTraffic)

	// 返回签到成功信息
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "签到成功",
		"data": gin.H{
			"reward_traffic":  formattedTraffic,
			"continuity_days": checkinLog.ContinuityDays,
			"checkin_date":    checkinLog.CheckinDate.Format("2006-01-02"),
		},
	})
}

// GetCheckinStatus 获取用户签到状态
func (h *UserCheckinHandler) GetCheckinStatus(c *gin.Context) {
	// 从Header中获取token并验证
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "请先登录"})
		return
	}

	// 验证token并获取用户信息
	user, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的认证信息"})
		return
	}

	// 检查用户状态
	if user.Status != 1 {
		c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "账户已被禁用"})
		return
	}

	// 检查用户今天是否已签到
	hasChecked, err := h.userCheckinService.HasCheckedToday(context.Background(), user.ID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取签到状态失败"})
		return
	}

	// 获取用户组信息
	group, err := h.userService.GetUserGroup(context.Background(), user.ID)
	if err != nil {
		h.logger.Error("获取用户组信息失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取用户组信息失败"})
		return
	}

	// 构建响应数据
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取成功",
		"data": gin.H{
			"has_checked":     hasChecked,
			"checkin_count":   user.CheckinCount,
			"continuity_days": user.ContinuityCheckin,
			"last_checkin": func() string {
				if user.LastCheckin == nil {
					return ""
				}
				return user.LastCheckin.Format("2006-01-02")
			}(),
			"min_reward": formatCheckinTraffic(group.CheckinMinTraffic),
			"max_reward": formatCheckinTraffic(group.CheckinMaxTraffic),
		},
	})
}

// GetCheckinLogs 获取用户签到记录
func (h *UserCheckinHandler) GetCheckinLogs(c *gin.Context) {
	// 从Header中获取token并验证
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "请先登录"})
		return
	}

	// 验证token并获取用户信息
	user, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的认证信息"})
		return
	}

	// 获取分页参数
	page := 1
	pageSize := 10

	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")

	if pageStr != "" {
		if pageNum, err := strconv.Atoi(pageStr); err == nil && pageNum > 0 {
			page = pageNum
		}
	}

	if pageSizeStr != "" {
		if pageSizeNum, err := strconv.Atoi(pageSizeStr); err == nil && pageSizeNum > 0 {
			if pageSizeNum > 50 {
				pageSizeNum = 50 // 限制最大为50条
			}
			pageSize = pageSizeNum
		}
	}

	// 获取签到记录
	logs, err := h.userCheckinService.GetCheckinLogs(context.Background(), user.ID, page, pageSize)
	if err != nil {
		h.logger.Error("获取签到记录失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取签到记录失败"})
		return
	}

	// 格式化签到记录
	formattedLogs := make([]map[string]interface{}, 0, len(logs))
	for _, log := range logs {
		formattedLogs = append(formattedLogs, map[string]interface{}{
			"id":              log.ID,
			"checkin_date":    log.CheckinDate.Format("2006-01-02"),
			"reward_traffic":  formatCheckinTraffic(log.RewardTraffic),
			"continuity_days": log.ContinuityDays,
			"created_at":      log.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	// 返回签到记录
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取成功",
		"data": gin.H{
			"logs": formattedLogs,
			"pagination": gin.H{
				"current_page": page,
				"page_size":    pageSize,
			},
		},
	})
}
