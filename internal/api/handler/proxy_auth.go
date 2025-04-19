package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

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
			RejectReason: "无效请求格式",
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
	token, _ := content["metas"].(map[string]interface{})["token"].(string)

	if username == "" {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "用户名不能为空",
		})
		return
	}

	// 从数据库获取用户信息
	user, err := h.userService.GetByUsername(context.Background(), username)
	if err != nil || user == nil {
		h.logger.Warn("用户不存在", "username", username)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "用户不存在或认证失败",
		})
		return
	}

	// 验证token：使用数据库中存储的token进行对比
	if user.Token != token {
		h.logger.Warn("用户Token不匹配", "username", username, "expected", user.Token, "actual", token)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "用户认证失败，Token无效",
		})
		return
	}

	// 验证用户状态
	if user.Status != 1 {
		h.logger.Warn("用户账号被禁用", "username", username)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "用户账号已被禁用",
		})
		return
	}

	// 登录成功
	h.logger.Info("用户登录成功", "username", username)
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
			RejectReason: "用户信息格式错误",
		})
		return
	}

	// 1. 提取用户信息并校验用户权限
	username, _ := userInfo["user"].(string)
	metas, metasOk := userInfo["metas"].(map[string]interface{})
	var token string
	if metasOk {
		token, _ = metas["token"].(string)
	}

	// 检查用户名和token
	if username == "" {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "用户名不能为空",
		})
		return
	}

	// 从数据库获取用户信息
	user, err := h.userService.GetByUsername(context.Background(), username)
	if err != nil || user == nil {
		h.logger.Warn("用户不存在", "username", username)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "用户不存在或认证失败",
		})
		return
	}

	// 验证token
	if user.Token != token {
		h.logger.Warn("用户Token不匹配", "username", username)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "用户认证失败，Token无效",
		})
		return
	}

	// 验证用户状态
	if user.Status != 1 {
		h.logger.Warn("用户账号被禁用", "username", username)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "用户账号已被禁用",
		})
		return
	}

	// 3. 解析隧道名称并校验格式
	fullProxyName, _ := content["proxy_name"].(string)
	if fullProxyName == "" {
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "隧道名称不能为空",
		})
		return
	}

	// 验证隧道名称格式：用户名.隧道名
	parts := strings.Split(fullProxyName, ".")
	if len(parts) != 2 || parts[0] != username {
		h.logger.Warn("隧道名称格式错误", "proxy_name", fullProxyName)
		c.JSON(http.StatusOK, FrpPluginResponse{
			Reject:       true,
			RejectReason: "隧道名称格式错误，应为：用户名.隧道名",
		})
		return
	}

	proxyName := parts[1]
	proxyType, _ := content["proxy_type"].(string)

	// 4. 根据隧道类型校验配置
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

		// 查询隧道信息，确认是否存在该隧道
		proxy, err := h.proxyService.GetByUsernameAndName(context.Background(), username, proxyName)
		if err != nil {
			h.logger.Error("查询隧道信息失败", "error", err)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "服务器内部错误",
			})
			return
		}

		// 如果隧道不存在，拒绝请求
		if proxy == nil {
			h.logger.Warn("隧道不存在", "username", username, "proxy_name", proxyName)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "隧道不存在",
			})
			return
		}

		// 检查隧道类型是否匹配
		if proxy.ProxyType != proxyType {
			h.logger.Warn("隧道类型不匹配", "expected", proxy.ProxyType, "actual", proxyType)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "隧道类型不匹配",
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

		// 检查用户是否有权限使用该节点
		hasAccess, err := h.proxyService.CheckUserNodeAccess(context.Background(), username, proxy.Node)
		if err != nil {
			h.logger.Error("检查节点权限失败", "error", err)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "服务器内部错误",
			})
			return
		}

		if !hasAccess {
			h.logger.Warn("用户无权使用此节点", "username", username, "node_id", proxy.Node)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "您无权使用此节点",
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

		// 查询隧道信息
		proxy, err := h.proxyService.GetByUsernameAndName(context.Background(), username, proxyName)
		if err != nil {
			h.logger.Error("查询隧道信息失败", "error", err)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "服务器内部错误",
			})
			return
		}

		// 如果隧道不存在，拒绝请求
		if proxy == nil {
			h.logger.Warn("隧道不存在", "username", username, "proxy_name", proxyName)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "隧道不存在",
			})
			return
		}

		// 检查隧道类型是否匹配
		if proxy.ProxyType != proxyType {
			h.logger.Warn("隧道类型不匹配", "expected", proxy.ProxyType, "actual", proxyType)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "隧道类型不匹配",
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

		// 检查用户是否有权限使用该节点
		hasAccess, err := h.proxyService.CheckUserNodeAccess(context.Background(), username, proxy.Node)
		if err != nil {
			h.logger.Error("检查节点权限失败", "error", err)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "服务器内部错误",
			})
			return
		}

		if !hasAccess {
			h.logger.Warn("用户无权使用此节点", "username", username, "node_id", proxy.Node)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "您无权使用此节点",
			})
			return
		}
	} else {
		// 其他类型的隧道，如stcp/tcpmux等
		proxy, err := h.proxyService.GetByUsernameAndName(context.Background(), username, proxyName)
		if err != nil {
			h.logger.Error("查询隧道信息失败", "error", err)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "服务器内部错误",
			})
			return
		}

		if proxy == nil {
			h.logger.Warn("隧道不存在", "username", username, "proxy_name", proxyName)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "隧道不存在",
			})
			return
		}

		if proxy.ProxyType != proxyType {
			h.logger.Warn("隧道类型不匹配", "expected", proxy.ProxyType, "actual", proxyType)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "隧道类型不匹配",
			})
			return
		}

		// 检查用户是否有权限使用该节点
		hasAccess, err := h.proxyService.CheckUserNodeAccess(context.Background(), username, proxy.Node)
		if err != nil {
			h.logger.Error("检查节点权限失败", "error", err)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "服务器内部错误",
			})
			return
		}

		if !hasAccess {
			h.logger.Warn("用户无权使用此节点", "username", username, "node_id", proxy.Node)
			c.JSON(http.StatusOK, FrpPluginResponse{
				Reject:       true,
				RejectReason: "您无权使用此节点",
			})
			return
		}
	}

	// 鉴权通过，更新隧道状态
	proxy, err := h.proxyService.GetByUsernameAndName(context.Background(), username, proxyName)
	if err == nil && proxy != nil {
		// 更新隧道状态为活跃
		proxy.Status = "active"
		proxy.LastUpdate = time.Now().Format("2006-01-02 15:04:05")
		if runID, ok := userInfo["run_id"].(string); ok {
			proxy.RunID = runID
		}

		err = h.proxyService.Update(context.Background(), proxy)
		if err != nil {
			h.logger.Error("更新隧道状态失败", "error", err)
		}
	}

	// 鉴权通过
	c.JSON(http.StatusOK, FrpPluginResponse{
		Reject:   false,
		Unchange: true,
	})
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
		proxy.Status = "inactive"
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
