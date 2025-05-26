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
	proxyService service.ProxyService
	userService  service.UserService
	logger       *logger.Logger
}

// NewProxyAuthHandler 创建隧道鉴权处理器实例
func NewProxyAuthHandler(proxyService service.ProxyService, userService service.UserService, logger *logger.Logger) *ProxyAuthHandler {
	return &ProxyAuthHandler{
		proxyService: proxyService,
		userService:  userService,
		logger:       logger,
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

	h.logger.Info("收到隧道鉴权请求", "op", req.Op, "version", req.Version)

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
	// 提取用户信息
	content := req.Content
	username, _ := content["user"].(string)

	// 记录完整的请求内容，帮助调试
	h.logger.Info("隧道登录请求详情", "content", content)

	// 获取token，并记录token获取过程
	var token string
	metas, hasMetas := content["metas"].(map[string]interface{})
	if hasMetas {
		tokenVal, hasToken := metas["token"]
		if hasToken {
			token, _ = tokenVal.(string)
			h.logger.Info("从metas中获取到token", "token_exists", token != "")
		} else {
			h.logger.Warn("metas中不存在token字段")
		}
	} else {
		h.logger.Warn("请求中不存在metas字段或格式错误")
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
		h.logger.Warn("用户不存在", "username", username, "error", err)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrAuthFailed,
		})
		return
	}

	h.logger.Info("用户信息获取成功", "username", username, "user_token_exists", user.Token != "")

	// 验证token：使用数据库中存储的token进行对比
	if user.Token != token {
		h.logger.Warn("用户Token不匹配",
			"username", username,
			"expected_token", user.Token,
			"actual_token", token,
			"token_length", len(token),
			"expected_length", len(user.Token))

		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrInvalidToken,
		})
		return
	}

	// 验证用户状态
	if user.Status != 1 {
		h.logger.Warn("用户账号被禁用", "username", username, "status", user.Status)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrAccountDisabled,
		})
		return
	}

	// 检查用户是否在黑名单中
	isBlacklisted, err := h.userService.IsUserBlacklistedByUsername(context.Background(), username)
	if err != nil {
		h.logger.Error("检查黑名单失败", "username", username, "error", err)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrInternalServer,
		})
		return
	}

	if isBlacklisted {
		h.logger.Warn("用户在黑名单中", "username", username)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrBlacklisted,
		})
		return
	}

	// 登录成功
	h.logger.Info("用户登录成功", "username", username, "group_id", user.GroupID)
	c.JSON(http.StatusOK, FrpPluginResponse{
		Reject:   false,
		Unchange: true,
	})
}

// handleNewProxyAuth 处理新隧道创建鉴权
func (h *ProxyAuthHandler) handleNewProxyAuth(c *gin.Context, req FrpPluginRequest) {
	content := req.Content

	// 记录完整的请求内容，帮助调试
	h.logger.Info("新隧道创建请求详情", "content", content)

	userInfo, ok := content["user"].(map[string]interface{})
	if !ok {
		h.logger.Warn("用户信息格式错误", "user_info_exists", content["user"] != nil)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrInvalidFormat,
		})
		return
	}

	// 1. 提取用户信息并校验用户权限
	username, _ := userInfo["user"].(string)

	// 获取token，并记录token获取过程
	var token string
	metas, metasOk := userInfo["metas"].(map[string]interface{})
	if metasOk {
		tokenVal, hasToken := metas["token"]
		if hasToken {
			token, _ = tokenVal.(string)
			h.logger.Info("从metas中获取到token", "token_exists", token != "", "token_length", len(token))
		} else {
			h.logger.Warn("metas中不存在token字段")
		}
	} else {
		h.logger.Warn("请求中不存在metas字段或格式错误")
	}

	// 检查用户名和token
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
		h.logger.Warn("用户不存在", "username", username, "error", err)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrAuthFailed,
		})
		return
	}

	h.logger.Info("用户信息获取成功", "username", username, "user_token_exists", user.Token != "", "token_length", len(user.Token))

	// 验证token
	if user.Token != token {
		h.logger.Warn("用户Token不匹配",
			"username", username,
			"expected_token", user.Token,
			"actual_token", token,
			"token_length", len(token),
			"expected_length", len(user.Token))

		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrInvalidToken,
		})
		return
	}

	// 验证用户状态
	if user.Status != 1 {
		h.logger.Warn("用户账号被禁用", "username", username, "status", user.Status)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrAccountDisabled,
		})
		return
	}

	// 检查用户是否在黑名单中
	isBlacklisted, err := h.userService.IsUserBlacklistedByUsername(context.Background(), username)
	if err != nil {
		h.logger.Error("检查黑名单失败", "username", username, "error", err)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrInternalServer,
		})
		return
	}

	if isBlacklisted {
		h.logger.Warn("用户在黑名单中", "username", username)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrBlacklisted,
		})
		return
	}

	// 3. 解析隧道名称并校验格式
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
		h.logger.Warn("隧道名称格式错误", "proxy_name", fullProxyName)
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

	// 如果隧道不存在，拒绝请求
	if proxy == nil {
		h.logger.Warn("隧道不存在", "username", username, "proxy_name", proxyName)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrProxyNotFound,
		})
		return
	}

	// 检查隧道类型是否匹配
	if proxy.ProxyType != proxyType {
		h.logger.Warn("隧道类型不匹配", "expected", proxy.ProxyType, "actual", proxyType)
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
		h.logger.Warn("用户无权使用此节点", "username", username, "node_id", proxy.Node)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: constants.ErrNoNodeAccess,
		})
		return
	}

	// 4. 根据隧道类型验证特定配置
	if proxyType == "tcp" || proxyType == "udp" {
		// 对于TCP/UDP类型，检查remote_port
		remotePort, ok := content["remote_port"].(float64)
		if !ok {
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "远程端口配置错误",
			})
			return
		}

		// 检查远程端口是否匹配
		remotePortStr := proxy.RemotePort
		if remotePortStr != "" && remotePortStr != strconv.Itoa(int(remotePort)) {
			h.logger.Warn("远程端口不匹配", "expected", remotePortStr, "actual", remotePort)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "远程端口不匹配",
			})
			return
		}
	} else if proxyType == "http" || proxyType == "https" {
		// 对于HTTP/HTTPS类型，检查custom_domains
		customDomains, ok := content["custom_domains"].([]interface{})
		if !ok || len(customDomains) == 0 {
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "域名配置错误",
			})
			return
		}

		// 检查域名是否匹配
		var domainFound bool
		for _, domain := range customDomains {
			if domainStr, ok := domain.(string); ok && domainStr == proxy.Domain {
				domainFound = true
				break
			}
		}

		if !domainFound {
			h.logger.Warn("域名不匹配", "expected", proxy.Domain)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "域名不匹配",
			})
			return
		}
	}

	// 5. 验证transport相关参数
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

	// 鉴权通过
	c.JSON(http.StatusOK, FrpPluginResponse{
		Reject:   false,
		Unchange: true,
	})
}

// verifyTransportParams 验证隧道的传输参数
func (h *ProxyAuthHandler) verifyTransportParams(c *gin.Context, content map[string]interface{}, proxy *repository.Proxy, user *repository.User) bool {
	// 获取传输参数
	useEncryption, hasEncryption := content["use_encryption"].(bool)
	bandwidthLimit, hasBandwidth := content["bandwidth_limit"].(string)
	bandwidthLimitMode, hasBandwidthMode := content["bandwidth_limit_mode"].(string)

	// 检查传输配置
	if hasEncryption && strconv.FormatBool(useEncryption) != proxy.UseEncryption {
		h.logger.Warn("加密设置不匹配", "expected", proxy.UseEncryption, "actual", useEncryption)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "加密设置不匹配",
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

	// 检查带宽限制 - 更灵活的比较方式
	if hasBandwidth {
		// 移除可能存在的引号
		cleanBandwidth := strings.Trim(bandwidthLimit, "\"")
		cleanExpected := expectedBandwidth

		if cleanBandwidth != cleanExpected {
			h.logger.Warn("带宽限制不匹配", "expected", expectedBandwidth, "actual", bandwidthLimit,
				"cleanExpected", cleanExpected, "cleanBandwidth", cleanBandwidth)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "带宽限制设置不匹配",
			})
			return false
		}
	}

	// 检查带宽限制模式
	if hasBandwidthMode && bandwidthLimitMode != "server" {
		h.logger.Warn("带宽限制模式不匹配", "expected", "server", "actual", bandwidthLimitMode)
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
	// 这里简单地允许关闭隧道操作
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
