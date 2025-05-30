package handler

import (
	"context"
	"fmt"
	"net/http"
	"stellarfrp/internal/constants"
	"strconv"
	"strings"
	"time"

	"stellarfrp/internal/repository"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"

	"github.com/gin-gonic/gin"
)

// FrpPluginRequest 定义FRP插件请求结构
type FrpPluginRequest struct {
	Version string                 `json:"version"`
	Op      string                 `json:"op"`
	Content map[string]interface{} `json:"content"`
}

// FrpPluginResponse 定义FRP插件响应结构
type FrpPluginResponse struct {
	Reject       bool                   `json:"reject,omitempty"`
	RejectReason string                 `json:"reject_reason,omitempty"`
	Unchange     bool                   `json:"unchange,omitempty"`
	Content      map[string]interface{} `json:"content,omitempty"`
}

// ProxyAuthHandler 隧道鉴权处理器
type ProxyAuthHandler struct {
	proxyService       service.ProxyService
	userService        service.UserService
	userTrafficService service.UserTrafficLogService
	logger             *logger.Logger
}

// NewProxyAuthHandler 创建隧道鉴权处理器实例
func NewProxyAuthHandler(proxyService service.ProxyService, userService service.UserService, userTrafficService service.UserTrafficLogService, logger *logger.Logger) *ProxyAuthHandler {
	return &ProxyAuthHandler{
		proxyService:       proxyService,
		userService:        userService,
		userTrafficService: userTrafficService,
		logger:             logger,
	}
}

// HandleProxyAuth 处理隧道鉴权请求
func (h *ProxyAuthHandler) HandleProxyAuth(c *gin.Context) {
	var req FrpPluginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("解析请求失败", "error", err)
		c.JSON(http.StatusBadRequest, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrInvalidRequest,
		})
		return
	}

	// 根据操作类型处理请求
	switch req.Op {
	case "Login":
		h.handleLoginAuth(c, req)
	case "NewProxy":
		h.handleNewProxyAuth(c, req)
	case "CloseProxy":
		h.handleCloseProxyAuth(c, req)
	default:
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "不支持的操作类型",
		})
	}
}

// handleLoginAuth 处理登录鉴权
func (h *ProxyAuthHandler) handleLoginAuth(c *gin.Context, req FrpPluginRequest) {
	content := req.Content
	username, _ := content["user"].(string)

	var token string
	metas, hasMetas := content["metas"].(map[string]interface{})
	if hasMetas {
		tokenVal, hasToken := metas["token"]
		if hasToken {
			token, _ = tokenVal.(string)
		}
	}

	if username == "" {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrUsernameEmpty,
		})
		return
	}

	// 从数据库获取用户信息
	user, err := h.userService.GetByUsername(context.Background(), username)
	if err != nil || user == nil {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrAuthFailed,
		})
		return
	}

	// 验证token
	if user.Token != token {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrInvalidToken,
		})
		return
	}

	// 验证用户状态
	if user.Status != 1 {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrAccountDisabled,
		})
		return
	}

	// 检查用户是否在黑名单中
	isBlacklisted, err := h.userService.IsUserBlacklistedByUsername(context.Background(), username)
	if err != nil {
		h.logger.Error("检查黑名单失败", "error", err)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrInternalServer,
		})
		return
	}

	if isBlacklisted {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrBlacklisted,
		})
		return
	}

	// 检查用户流量是否超额
	userTrafficLog, err := h.userTrafficService.GetUserTodayTraffic(context.Background(), username)
	if err != nil {
		h.logger.Error("获取用户今日流量失败", "error", err)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrInternalServer,
		})
		return
	}

	totalTrafficQuotaBytes := int64(0)
	userGroup, err := h.userService.GetUserGroup(context.Background(), user.ID)
	if err != nil {
		h.logger.Error("获取用户组信息失败", "error", err)
	} else if userGroup != nil {
		totalTrafficQuotaBytes += userGroup.TrafficQuota
	}

	if user.TrafficQuota != nil {
		totalTrafficQuotaBytes += *user.TrafficQuota
	}

	if totalTrafficQuotaBytes > 0 && userTrafficLog.TotalTraffic >= totalTrafficQuotaBytes {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrTrafficExhausted,
		})
		return
	}

	c.JSON(http.StatusOK, FrpPluginResponse{
		Reject:   false,
		Unchange: true,
	})
}

// handleNewProxyAuth 处理新隧道创建鉴权
func (h *ProxyAuthHandler) handleNewProxyAuth(c *gin.Context, req FrpPluginRequest) {
	content := req.Content

	userInfo, ok := content["user"].(map[string]interface{})
	if !ok {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrInvalidFormat,
		})
		return
	}

	username, _ := userInfo["user"].(string)

	var token string
	metas, metasOk := userInfo["metas"].(map[string]interface{})
	if metasOk {
		tokenVal, hasToken := metas["token"]
		if hasToken {
			token, _ = tokenVal.(string)
		}
	}

	if username == "" {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrUsernameEmpty,
		})
		return
	}

	// 从数据库获取用户信息
	user, err := h.userService.GetByUsername(context.Background(), username)
	if err != nil || user == nil {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrAuthFailed,
		})
		return
	}

	// 验证token
	if user.Token != token {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrInvalidToken,
		})
		return
	}

	// 验证用户状态
	if user.Status != 1 {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrAccountDisabled,
		})
		return
	}

	// 检查用户是否在黑名单中
	isBlacklisted, err := h.userService.IsUserBlacklistedByUsername(context.Background(), username)
	if err != nil {
		h.logger.Error("检查黑名单失败", "error", err)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrInternalServer,
		})
		return
	}

	if isBlacklisted {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrBlacklisted,
		})
		return
	}

	// 解析隧道名称并校验格式
	fullProxyName, _ := content["proxy_name"].(string)
	if fullProxyName == "" {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrProxyNameEmpty,
		})
		return
	}

	// 验证隧道名称格式：用户名.隧道名
	parts := strings.Split(fullProxyName, ".")
	if len(parts) != 2 || parts[0] != username {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrProxyNameFormat,
		})
		return
	}

	proxyName := parts[1]
	proxyType, _ := content["proxy_type"].(string)

	// 获取隧道信息
	proxy, err := h.proxyService.GetByUsernameAndName(context.Background(), username, proxyName)
	if err != nil {
		h.logger.Error("查询隧道信息失败", "error", err)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrInternalServer,
		})
		return
	}

	if proxy == nil {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrProxyNotFound,
		})
		return
	}

	if proxy.ProxyType != proxyType {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrProxyTypeMismatch,
		})
		return
	}

	// 检查用户是否有权限使用该节点
	hasAccess, err := h.proxyService.CheckUserNodeAccess(context.Background(), username, proxy.Node)
	if err != nil {
		h.logger.Error("检查节点权限失败", "error", err)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrInternalServer,
		})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrNoNodeAccess,
		})
		return
	}

	if !h.verifyTransportParams(c, content, proxy, user) {
		return
	}

	// 鉴权通过，更新隧道状态
	proxy.Status = "online"
	proxy.LastUpdate = time.Now().Format("2006-01-02 15:04:05")
	if runID, ok := userInfo["run_id"].(string); ok {
		proxy.RunID = runID
	}

	err = h.proxyService.Update(context.Background(), proxy)
	if err != nil {
		h.logger.Error("更新隧道状态失败", "error", err)
	}

	c.JSON(http.StatusOK, FrpPluginResponse{
		Reject:   false,
		Unchange: true,
	})
}

// verifyTransportParams 验证隧道的传输参数
func (h *ProxyAuthHandler) verifyTransportParams(c *gin.Context, content map[string]interface{}, proxy *repository.Proxy, user *repository.User) bool {
	// 获取传输参数
	useEncryption, hasEncryption := content["use_encryption"].(bool)
	useCompression, hasCompression := content["use_compression"].(bool)
	bandwidthLimit, hasBandwidth := content["bandwidth_limit"].(string)
	bandwidthLimitMode, hasBandwidthMode := content["bandwidth_limit_mode"].(string)

	// 检查参数是否存在
	if !hasEncryption || !hasCompression || !hasBandwidth || !hasBandwidthMode {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "非法传参",
		})
		return false
	}

	// 检查传输配置
	if strconv.FormatBool(useEncryption) != proxy.UseEncryption {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "非法传参",
		})
		return false
	}

	// 检查压缩配置
	if strconv.FormatBool(useCompression) != proxy.UseCompression {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "非法传参",
		})
		return false
	}

	// 获取用户带宽限制
	userGroup, err := h.userService.GetUserGroup(context.Background(), user.ID)
	if err != nil {
		h.logger.Error("获取用户组信息失败", "error", err)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "服务器内部错误",
		})
		return false
	}

	// 计算带宽限制
	userBandwidth := 0
	if user.Bandwidth != nil {
		userBandwidth = *user.Bandwidth
	}
	totalBandwidth := userGroup.BandwidthLimit + userBandwidth
	expectedBandwidth := fmt.Sprintf("%d", totalBandwidth) + "MB"

	// 检查带宽限制
	cleanBandwidth := strings.Trim(bandwidthLimit, "\"")
	cleanExpected := expectedBandwidth

	if cleanBandwidth != cleanExpected {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "带宽限制设置不匹配",
		})
		return false
	}

	// 检查带宽限制模式
	if bandwidthLimitMode != "server" {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "带宽限制模式必须为server",
		})
		return false
	}

	return true
}

// handleCloseProxyAuth 处理关闭隧道鉴权
func (h *ProxyAuthHandler) handleCloseProxyAuth(c *gin.Context, req FrpPluginRequest) {
	content := req.Content
	userInfo, ok := content["user"].(map[string]interface{})
	if !ok {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "用户信息格式错误",
		})
		return
	}

	username, _ := userInfo["user"].(string)
	proxyName, _ := content["proxy_name"].(string)

	if username == "" || proxyName == "" {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "用户名或隧道名称不能为空",
		})
		return
	}

	// 解析隧道名称，可能是 username.proxyname 格式
	parts := strings.Split(proxyName, ".")
	if len(parts) == 2 {
		proxyName = parts[1]
	}

	// 更新隧道状态为非活跃
	proxy, err := h.proxyService.GetByUsernameAndName(context.Background(), username, proxyName)
	if err == nil && proxy != nil {
		proxy.Status = "offline"
		proxy.LastUpdate = time.Now().Format("2006-01-02 15:04:05")
		proxy.RunID = ""

		err = h.proxyService.Update(context.Background(), proxy)
		if err != nil {
			h.logger.Error("更新隧道状态失败", "error", err)
		}
	}

	c.JSON(http.StatusOK, FrpPluginResponse{
		Reject:   false,
		Unchange: true,
	})
}
