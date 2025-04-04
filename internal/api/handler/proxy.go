package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"stellarfrp/internal/repository"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ProxyHandler 隧道处理器
type ProxyHandler struct {
	proxyService service.ProxyService
	nodeService  service.NodeService
	userService  service.UserService
	logger       *logger.Logger
}

// NewProxyHandler 创建隧道处理器实例
func NewProxyHandler(proxyService service.ProxyService, nodeService service.NodeService, userService service.UserService, logger *logger.Logger) *ProxyHandler {
	return &ProxyHandler{
		proxyService: proxyService,
		nodeService:  nodeService,
		userService:  userService,
		logger:       logger,
	}
}

// CreateProxy 创建隧道
func (h *ProxyHandler) CreateProxy(c *gin.Context) {
	// 从请求头获取token
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未授权，请先登录"})
		return
	}

	// 通过token获取用户信息
	user, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的token"})
		return
	}

	// 解析请求参数
	type ProxyRequest struct {
		NodeID               int64  `json:"nodeId" binding:"required"`
		ProxyName            string `json:"proxyName" binding:"required"`
		LocalIP              string `json:"localIp" binding:"required"`
		LocalPort            int    `json:"localPort" binding:"required"`
		RemotePort           int    `json:"remotePort"`
		Domain               string `json:"domain"`
		ProxyType            string `json:"proxyType" binding:"required"`
		HostHeaderRewrite    string `json:"hostHeaderRewrite"`
		HeaderXFromWhere     string `json:"headerXFromWhere"`
		ProxyProtocolVersion string `json:"proxyProtocolVersion"`
		UseEncryption        bool   `json:"useEncryption"`
		UseCompression       bool   `json:"useCompression"`
	}

	var req ProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 获取节点信息
	node, err := h.nodeService.GetByID(context.Background(), req.NodeID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "节点不存在或已下线"})
		return
	}

	// 检查用户是否有权限访问该节点
	nodes, err := h.nodeService.GetAccessibleNodes(context.Background(), user.GroupID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点权限失败"})
		return
	}

	// 检查节点是否在用户可访问的节点列表中
	nodeAccessible := false
	for _, n := range nodes {
		if n.ID == req.NodeID {
			nodeAccessible = true
			break
		}
	}

	if !nodeAccessible {
		c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "您没有权限使用该节点"})
		return
	}

	// 检查隧道类型是否被节点支持
	var allowedTypes []string
	if err := json.Unmarshal([]byte(node.AllowedTypes), &allowedTypes); err != nil {
		h.logger.Error("Failed to parse allowed_types", "error", err, "value", node.AllowedTypes)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	typeAllowed := false
	for _, t := range allowedTypes {
		if strings.EqualFold(t, req.ProxyType) {
			typeAllowed = true
			break
		}
	}

	if !typeAllowed {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "该节点不支持 " + req.ProxyType + " 类型的隧道"})
		return
	}

	// 检查同一节点下相同协议类型的隧道是否已经使用了相同的远程端口
	if req.RemotePort != 0 {
		isUsed, err := h.proxyService.IsRemotePortUsed(context.Background(), req.NodeID, req.ProxyType, strconv.Itoa(req.RemotePort))
		if err != nil {
			h.logger.Error("Failed to check remote port usage", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "检查端口占用失败"})
			return
		}
		if isUsed {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "该节点下已有相同协议类型的隧道使用了端口 " + strconv.Itoa(req.RemotePort) + "，请更换端口"})
			return
		}
	}

	// 根据隧道类型进行不同的验证
	if req.ProxyType == "http" || req.ProxyType == "https" {
		// HTTP/HTTPS类型必须填写domain
		if req.Domain == "" {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "HTTP/HTTPS类型的隧道必须填写域名"})
			return
		}

		// HTTP类型remotePort必须为80，HTTPS必须为443
		expectedPort := 80
		if req.ProxyType == "https" {
			expectedPort = 443
		}
		if req.RemotePort != expectedPort {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": req.ProxyType + "类型的隧道远程端口必须为" + strconv.Itoa(expectedPort)})
			return
		}
	} else if req.ProxyType == "tcp" || req.ProxyType == "udp" {
		// TCP/UDP类型需要验证remotePort是否在节点的port_range范围内
		if req.RemotePort == 0 {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "TCP/UDP类型的隧道必须填写远程端口"})
			return
		}

		// 解析port_range (格式通常为"10000-25555")
		portRange := strings.Split(node.PortRange, "-")
		if len(portRange) != 2 {
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "节点端口范围配置错误"})
			return
		}

		minPort, err1 := strconv.Atoi(portRange[0])
		maxPort, err2 := strconv.Atoi(portRange[1])
		remotePort := req.RemotePort

		if err1 != nil || err2 != nil {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "端口格式错误"})
			return
		}

		if remotePort < minPort || remotePort > maxPort {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "远程端口必须在" + node.PortRange + "范围内"})
			return
		}
	}

	// 检查用户是否已有同名隧道
	existingProxy, err := h.proxyService.GetByUsernameAndName(context.Background(), user.Username, req.ProxyName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		h.logger.Error("Failed to check existing proxy", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	if existingProxy != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "您已有同名隧道，请更换隧道名称"})
		return
	}

	// 获取用户当前的隧道数量
	proxyCount, err := h.proxyService.GetUserProxyCount(context.Background(), user.Username)
	if err != nil {
		h.logger.Error("Failed to get user proxy count", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取隧道数量失败"})
		return
	}

	// 获取用户组的隧道数量限制
	tunnelLimit, err := h.userService.GetGroupTunnelLimit(context.Background(), user.GroupID)
	if err != nil {
		h.logger.Error("Failed to get group tunnel limit", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取用户组隧道限制失败"})
		return
	}

	// 获取用户的附加隧道数量
	additionalTunnels := 0
	if user.TunnelCount != nil {
		additionalTunnels = *user.TunnelCount
	}

	// 计算总的隧道数量限制：用户组限制 + 用户附加隧道数量
	totalTunnelLimit := tunnelLimit + additionalTunnels

	// 检查是否达到上限
	if proxyCount >= totalTunnelLimit {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "您已达到可创建的数量上限"})
		return
	}

	// 创建隧道对象
	proxy := &repository.Proxy{
		Username:          user.Username,
		ProxyName:         req.ProxyName,
		ProxyType:         req.ProxyType,
		LocalIP:           req.LocalIP,
		LocalPort:         req.LocalPort,
		UseEncryption:     strconv.FormatBool(req.UseEncryption),
		UseCompression:    strconv.FormatBool(req.UseCompression),
		Domain:            req.Domain,
		HostHeaderRewrite: req.HostHeaderRewrite,
		RemotePort:        strconv.Itoa(req.RemotePort),
		HeaderXFromWhere:  req.HeaderXFromWhere,
		Node:              req.NodeID,
		Status:            "inactive", // 默认为未激活状态
	}

	// 创建隧道
	id, err := h.proxyService.Create(context.Background(), proxy)
	if err != nil {
		h.logger.Error("Failed to create proxy", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "创建隧道失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "创建成功", "data": gin.H{"Id": id}})
}

// UpdateProxy 更新隧道
func (h *ProxyHandler) UpdateProxy(c *gin.Context) {
	// 从请求头获取token
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未授权，请先登录"})
		return
	}

	// 通过token获取用户信息
	user, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的token"})
		return
	}

	// 解析请求参数
	type ProxyRequest struct {
		ID                   int64  `json:"id" binding:"required"`
		NodeID               int64  `json:"nodeId" binding:"required"`
		ProxyName            string `json:"proxyName" binding:"required"`
		LocalIP              string `json:"localIp" binding:"required"`
		LocalPort            int    `json:"localPort" binding:"required"`
		RemotePort           int    `json:"remotePort"`
		Domain               string `json:"domain"`
		ProxyType            string `json:"proxyType" binding:"required"`
		HostHeaderRewrite    string `json:"hostHeaderRewrite"`
		HeaderXFromWhere     string `json:"headerXFromWhere"`
		ProxyProtocolVersion string `json:"proxyProtocolVersion"`
		UseEncryption        bool   `json:"useEncryption"`
		UseCompression       bool   `json:"useCompression"`
	}

	var req ProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 检查隧道是否存在
	existingProxy, err := h.proxyService.GetByID(context.Background(), req.ID)
	if err != nil {
		h.logger.Error("Failed to check proxy existence", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "检查隧道失败: " + err.Error()})
		return
	}
	if existingProxy == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "隧道不存在"})
		return
	}

	// 检查用户是否有权限修改该隧道（只能修改自己的隧道）
	if existingProxy.Username != user.Username {
		c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "您没有权限修改此隧道"})
		return
	}

	// 获取节点信息
	node, err := h.nodeService.GetByID(context.Background(), req.NodeID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "节点不存在或已下线"})
		return
	}

	// 检查当前隧道是否属于请求中指定的节点
	if existingProxy.Node != req.NodeID {
		// 获取原节点信息
		originalNode, err := h.nodeService.GetByID(context.Background(), existingProxy.Node)
		if err == nil {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "该隧道属于 " + originalNode.NodeName + " 节点，不能修改为其他节点"})
		} else {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "该隧道不属于请求中指定的节点"})
		}
		return
	}

	// 检查用户是否有权限访问该节点
	nodes, err := h.nodeService.GetAccessibleNodes(context.Background(), user.GroupID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点权限失败"})
		return
	}

	// 检查节点是否在用户可访问的节点列表中
	nodeAccessible := false
	for _, n := range nodes {
		if n.ID == req.NodeID {
			nodeAccessible = true
			break
		}
	}

	if !nodeAccessible {
		c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "您没有权限使用该节点"})
		return
	}

	// 检查隧道类型是否被节点支持
	var allowedTypes []string
	if err := json.Unmarshal([]byte(node.AllowedTypes), &allowedTypes); err != nil {
		h.logger.Error("Failed to parse allowed_types", "error", err, "value", node.AllowedTypes)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	typeAllowed := false
	for _, t := range allowedTypes {
		if strings.EqualFold(t, req.ProxyType) {
			typeAllowed = true
			break
		}
	}

	if !typeAllowed {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "该节点不支持 " + req.ProxyType + " 类型的隧道"})
		return
	}

	// 检查同一节点下相同协议类型的隧道是否已经使用了相同的远程端口（排除当前正在编辑的隧道）
	if req.RemotePort != 0 {
		isUsed, err := h.proxyService.IsRemotePortUsed(context.Background(), req.NodeID, req.ProxyType, strconv.Itoa(req.RemotePort))
		if err != nil {
			h.logger.Error("Failed to check remote port usage", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "检查端口占用失败"})
			return
		}
		// 如果端口被占用，但不是被当前编辑的隧道占用，则返回错误
		if isUsed && (existingProxy.RemotePort != strconv.Itoa(req.RemotePort) || existingProxy.Node != req.NodeID) {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "该节点下已有相同协议类型的隧道使用了端口 " + strconv.Itoa(req.RemotePort) + "，请更换端口"})
			return
		}
	}

	// 根据隧道类型进行不同的验证
	if req.ProxyType == "http" || req.ProxyType == "https" {
		// HTTP/HTTPS类型必须填写domain
		if req.Domain == "" {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "HTTP/HTTPS类型的隧道必须填写域名"})
			return
		}

		// HTTP类型remotePort必须为80，HTTPS必须为443
		expectedPort := 80
		if req.ProxyType == "https" {
			expectedPort = 443
		}
		if req.RemotePort != expectedPort {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": req.ProxyType + "类型的隧道远程端口必须为" + strconv.Itoa(expectedPort)})
			return
		}
	} else if req.ProxyType == "tcp" || req.ProxyType == "udp" {
		// TCP/UDP类型需要验证remotePort是否在节点的port_range范围内
		if req.RemotePort == 0 {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "TCP/UDP类型的隧道必须填写远程端口"})
			return
		}

		// 解析port_range (格式通常为"10000-25555")
		portRange := strings.Split(node.PortRange, "-")
		if len(portRange) != 2 {
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "节点端口范围配置错误"})
			return
		}

		minPort, err1 := strconv.Atoi(portRange[0])
		maxPort, err2 := strconv.Atoi(portRange[1])
		remotePort := req.RemotePort

		if err1 != nil || err2 != nil {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "端口格式错误"})
			return
		}

		if remotePort < minPort || remotePort > maxPort {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "远程端口必须在" + node.PortRange + "范围内"})
			return
		}
	}

	// 检查隧道名称是否已被其他隧道使用（不包括当前隧道）
	if existingProxy.ProxyName != req.ProxyName {
		otherProxy, err := h.proxyService.GetByUsernameAndName(context.Background(), user.Username, req.ProxyName)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			h.logger.Error("Failed to check existing proxy", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
			return
		}
		if otherProxy != nil {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "您已有同名隧道，请更换隧道名称"})
			return
		}
	}

	// 更新隧道对象
	proxy := &repository.Proxy{
		ID:                req.ID,
		Username:          user.Username,
		ProxyName:         req.ProxyName,
		ProxyType:         req.ProxyType,
		LocalIP:           req.LocalIP,
		LocalPort:         req.LocalPort,
		UseEncryption:     strconv.FormatBool(req.UseEncryption),
		UseCompression:    strconv.FormatBool(req.UseCompression),
		Domain:            req.Domain,
		HostHeaderRewrite: req.HostHeaderRewrite,
		RemotePort:        strconv.Itoa(req.RemotePort),
		HeaderXFromWhere:  req.HeaderXFromWhere,
		Node:              req.NodeID,
		Status:            existingProxy.Status, // 保持原有状态
	}

	// 更新隧道
	err = h.proxyService.Update(context.Background(), proxy)
	if err != nil {
		h.logger.Error("Failed to update proxy", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "更新隧道失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "更新成功", "data": gin.H{"id": req.ID}})
}

// DeleteProxy 删除隧道
func (h *ProxyHandler) DeleteProxy(c *gin.Context) {
	// 从请求头获取token
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未授权，请先登录"})
		return
	}

	// 通过token获取用户信息
	user, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的token"})
		return
	}

	// 解析请求参数
	type DeleteRequest struct {
		ID int64 `json:"id" binding:"required"`
	}

	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 检查隧道是否存在
	proxy, err := h.proxyService.GetByID(context.Background(), req.ID)
	if err != nil {
		h.logger.Error("Failed to check proxy existence", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "检查隧道失败: " + err.Error()})
		return
	}
	if proxy == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "隧道不存在"})
		return
	}

	// 检查用户是否有权限删除该隧道（只能删除自己的隧道）
	if proxy.Username != user.Username {
		c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "您没有权限删除此隧道"})
		return
	}

	// 删除隧道
	err = h.proxyService.Delete(context.Background(), req.ID)
	if err != nil {
		h.logger.Error("Failed to delete proxy", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "删除隧道失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "删除成功"})
}

// GetProxyByID 根据ID获取隧道
func (h *ProxyHandler) GetProxyByID(c *gin.Context) {
	// 从请求头获取token
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未授权，请先登录"})
		return
	}

	// 通过token获取用户信息
	user, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的token"})
		return
	}

	// 解析请求参数
	type GetProxyRequest struct {
		ID int64 `json:"id"`
	}

	var req GetProxyRequest
	// 尝试解析JSON请求体，如果为空或格式错误，则默认返回所有隧道
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		// 只有在不是EOF错误时才返回错误信息
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 如果提供了ID，则获取单个隧道
	if req.ID > 0 {
		proxy, err := h.proxyService.GetByID(context.Background(), req.ID)
		if err != nil {
			h.logger.Error("Failed to get proxy", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取隧道失败: " + err.Error()})
			return
		}

		if proxy == nil {
			c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "隧道不存在"})
			return
		}

		// 检查是否是用户自己的隧道
		if proxy.Username != user.Username {
			c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "您没有权限查看此隧道"})
			return
		}

		// 获取节点信息
		node, err := h.nodeService.GetByID(context.Background(), proxy.Node)
		if err != nil {
			h.logger.Error("Failed to get node", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点信息失败"})
			return
		}

		// 构建Link
		remotePort, _ := strconv.Atoi(proxy.RemotePort)
		link := ""
		if node.Host.Valid && node.Host.String != "" {
			link = node.Host.String + ":" + proxy.RemotePort
		} else {
			link = node.IP + ":" + proxy.RemotePort
		}

		// 根据隧道类型生成不同的data内容
		data := ""
		// 基本配置信息
		baseConfig := "serverAddr = \"" + node.IP + "\"\n" +
			"serverPort = " + strconv.Itoa(node.FrpsPort) + "\n" +
			"user = \"" + proxy.Username + "\"\n" +
			"metadatas.token = \"" + user.Token + "\"\n"

		switch proxy.ProxyType {
		case "http":
			// HTTP类型隧道配置
			data = baseConfig +
				"[[proxies]]\n" +
				"name = \"" + proxy.ProxyName + "\"\n" +
				"type = \"http\"\n" +
				"localIP = \"" + proxy.LocalIP + "\"\n" +
				"localPort = " + strconv.Itoa(proxy.LocalPort) + "\n"

			// 添加域名信息
			if proxy.Domain != "" {
				data += "customDomains = [\"" + proxy.Domain + "\"]\n"
			}

		case "https":
			// HTTPS类型隧道配置
			data = baseConfig +
				"[[proxies]]\n" +
				"name = \"" + proxy.ProxyName + "\"\n" +
				"type = \"https\"\n"

			// 添加域名信息
			if proxy.Domain != "" {
				data += "customDomains = [\"" + proxy.Domain + "\"]\n"
			}

			// HTTPS插件配置
			data += "[proxies.plugin]\n" +
				"type = \"https2http\"\n" +
				"localAddr = \"" + proxy.LocalIP + ":" + strconv.Itoa(proxy.LocalPort) + "\"\n" +
				"# HTTPS 证书相关的配置\n" +
				"crtPath = \"./server.crt\"\n" +
				"keyPath = \"./server.key\"\n"

			// 添加主机头重写(如果有)
			if proxy.HostHeaderRewrite != "" {
				data += "hostHeaderRewrite = \"" + proxy.HostHeaderRewrite + "\"\n"
			}

			// 添加请求头信息(如果有)
			if proxy.HeaderXFromWhere != "" {
				data += "requestHeaders.set.x-from-where = \"" + proxy.HeaderXFromWhere + "\"\n"
			}

		default: // tcp, udp 和其他类型
			// 标准配置
			data = baseConfig +
				"[[proxies]]\n" +
				"name = \"" + proxy.ProxyName + "\"\n" +
				"type = \"" + proxy.ProxyType + "\"\n" +
				"localIP = \"" + proxy.LocalIP + "\"\n" +
				"localPort = " + strconv.Itoa(proxy.LocalPort) + "\n" +
				"remotePort = " + proxy.RemotePort + "\n"
		}

		// 构建返回数据
		tunnelData := gin.H{
			"Id":         proxy.ID,
			"ProxyName":  proxy.ProxyName,
			"ProxyType":  proxy.ProxyType,
			"LocalIp":    proxy.LocalIP,
			"LocalPort":  proxy.LocalPort,
			"RemotePort": remotePort,
			"Domains":    proxy.Domain,
			"Status":     proxy.Status,
			"NodeName":   node.NodeName,
			"Link":       link,
			"Type":       proxy.ProxyType,
			"Timestamp":  proxy.LastUpdate,
			"data":       data,
		}

		c.JSON(http.StatusOK, gin.H{
			"code":   200,
			"msg":    "获取成功",
			"tunnel": gin.H{strconv.FormatInt(proxy.ID, 10): tunnelData},
		})
		return
	}

	// 如果没有提供ID，则获取用户的所有隧道
	proxies, err := h.proxyService.GetByUsername(context.Background(), user.Username)
	if err != nil {
		h.logger.Error("Failed to get proxies", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取隧道列表失败: " + err.Error()})
		return
	}

	// 构建返回数据
	tunnels := make(gin.H)

	for _, proxy := range proxies {
		// 获取节点信息
		node, err := h.nodeService.GetByID(context.Background(), proxy.Node)
		if err != nil {
			h.logger.Warn("Failed to get node info for proxy", "proxyID", proxy.ID, "nodeID", proxy.Node, "error", err)
			continue
		}

		// 构建Link
		remotePort, _ := strconv.Atoi(proxy.RemotePort)
		link := ""
		if node.Host.Valid && node.Host.String != "" {
			link = node.Host.String + ":" + proxy.RemotePort
		} else {
			link = node.IP + ":" + proxy.RemotePort
		}

		// 根据隧道类型生成不同的data内容
		data := ""
		// 基本配置信息
		baseConfig := "serverAddr = \"" + node.IP + "\"\n" +
			"serverPort = " + strconv.Itoa(node.FrpsPort) + "\n" +
			"user = \"" + proxy.Username + "\"\n" +
			"metadatas.token = \"" + user.Token + "\"\n"

		switch proxy.ProxyType {
		case "http":
			// HTTP类型隧道配置
			data = baseConfig +
				"[[proxies]]\n" +
				"name = \"" + proxy.ProxyName + "\"\n" +
				"type = \"http\"\n" +
				"localIP = \"" + proxy.LocalIP + "\"\n" +
				"localPort = " + strconv.Itoa(proxy.LocalPort) + "\n"

			// 添加域名信息
			if proxy.Domain != "" {
				data += "customDomains = [\"" + proxy.Domain + "\"]\n"
			}

		case "https":
			// HTTPS类型隧道配置
			data = baseConfig +
				"[[proxies]]\n" +
				"name = \"" + proxy.ProxyName + "\"\n" +
				"type = \"https\"\n"

			// 添加域名信息
			if proxy.Domain != "" {
				data += "customDomains = [\"" + proxy.Domain + "\"]\n"
			}

			// HTTPS插件配置
			data += "[proxies.plugin]\n" +
				"type = \"https2http\"\n" +
				"localAddr = \"" + proxy.LocalIP + ":" + strconv.Itoa(proxy.LocalPort) + "\"\n" +
				"# HTTPS 证书相关的配置\n" +
				"crtPath = \"./server.crt\"\n" +
				"keyPath = \"./server.key\"\n"

			// 添加主机头重写(如果有)
			if proxy.HostHeaderRewrite != "" {
				data += "hostHeaderRewrite = \"" + proxy.HostHeaderRewrite + "\"\n"
			}

			// 添加请求头信息(如果有)
			if proxy.HeaderXFromWhere != "" {
				data += "requestHeaders.set.x-from-where = \"" + proxy.HeaderXFromWhere + "\"\n"
			}

		default: // tcp, udp 和其他类型
			// 标准配置
			data = baseConfig +
				"[[proxies]]\n" +
				"name = \"" + proxy.ProxyName + "\"\n" +
				"type = \"" + proxy.ProxyType + "\"\n" +
				"localIP = \"" + proxy.LocalIP + "\"\n" +
				"localPort = " + strconv.Itoa(proxy.LocalPort) + "\n" +
				"remotePort = " + proxy.RemotePort + "\n"
		}

		// 添加隧道信息
		tunnels[strconv.FormatInt(proxy.ID, 10)] = gin.H{
			"Id":         proxy.ID,
			"ProxyName":  proxy.ProxyName,
			"ProxyType":  proxy.ProxyType,
			"LocalIp":    proxy.LocalIP,
			"LocalPort":  proxy.LocalPort,
			"RemotePort": remotePort,
			"Domains":    proxy.Domain,
			"Status":     proxy.Status,
			"NodeName":   node.NodeName,
			"Link":       link,
			"Type":       proxy.ProxyType,
			"Timestamp":  proxy.LastUpdate,
			"data":       data,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"msg":    "获取成功",
		"tunnel": tunnels,
	})
}

// GetProxyStatus 获取隧道状态
func (h *ProxyHandler) GetProxyStatus(c *gin.Context) {
	// 从请求头获取token
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未授权，请先登录"})
		return
	}

	// 通过token获取用户信息
	user, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的token"})
		return
	}

	// 解析请求参数
	type StatusRequest struct {
		IDs []string `json:"id"`
	}

	var reqData StatusRequest
	// 解析JSON请求体，如果为空或格式错误，默认获取所有隧道
	if err := c.ShouldBindJSON(&reqData); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 如果ids为空，则获取用户所有隧道
	if len(reqData.IDs) == 0 {
		// 获取用户的所有隧道
		proxies, err := h.proxyService.GetByUsername(context.Background(), user.Username)
		if err != nil {
			h.logger.Error("Failed to get user's proxies", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取用户隧道列表失败"})
			return
		}

		// 如果用户没有隧道
		if len(proxies) == 0 {
			c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "获取成功", "data": gin.H{}})
			return
		}

		// 构建隧道ID列表
		reqData.IDs = make([]string, len(proxies))
		for i, proxy := range proxies {
			reqData.IDs[i] = strconv.FormatInt(proxy.ID, 10)
		}
	}

	// 验证所有隧道是否属于该用户并获取对应节点信息
	proxyNodeMap := make(map[int64]int64)       // 隧道ID -> 节点ID
	proxyNameMap := make(map[int64]string)      // 隧道ID -> 隧道名称
	nodeNameMap := make(map[int64]string)       // 节点ID -> 节点名称
	nodeMap := make(map[int64]*repository.Node) // 节点ID -> 节点信息

	for _, idStr := range reqData.IDs {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			h.logger.Error("Invalid proxy ID", "id", idStr, "error", err)
			continue
		}

		proxy, err := h.proxyService.GetByID(context.Background(), id)
		if err != nil {
			h.logger.Error("Failed to get proxy", "id", id, "error", err)
			continue
		}

		if proxy == nil {
			continue // 隧道不存在，跳过
		}

		// 验证隧道是否属于该用户
		if proxy.Username != user.Username {
			continue // 隧道不属于该用户，跳过
		}

		// 获取节点信息
		node, err := h.nodeService.GetByID(context.Background(), proxy.Node)
		if err != nil {
			h.logger.Error("Failed to get node", "nodeID", proxy.Node, "error", err)
			continue
		}

		// 记录节点名称和节点信息
		nodeNameMap[proxy.Node] = node.NodeName
		nodeMap[proxy.Node] = node

		// 记录隧道ID和对应的节点ID
		proxyNodeMap[proxy.ID] = proxy.Node
		proxyNameMap[proxy.ID] = proxy.ProxyName
	}

	if len(proxyNodeMap) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "未找到有效的隧道"})
		return
	}

	// 创建HTTP客户端，设置超时
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 收集结果
	var mu sync.Mutex
	results := make(map[string]gin.H)

	// 根据节点对隧道进行分组
	nodeProxiesMap := make(map[int64][]int64) // 节点ID -> 隧道ID列表

	for proxyID, nodeID := range proxyNodeMap {
		nodeProxiesMap[nodeID] = append(nodeProxiesMap[nodeID], proxyID)
	}

	// 多个隧道，使用并发请求
	var wg sync.WaitGroup

	// 并发请求每个节点的API来获取所有类型的隧道状态
	for nodeID, proxyIDs := range nodeProxiesMap {
		wg.Add(1)
		go func(nodeID int64, proxyIDs []int64) {
			defer wg.Done()

			node := nodeMap[nodeID]
			nodeName := nodeNameMap[nodeID]

			// 构建节点API URL - 请求所有隧道状态
			apiURL := fmt.Sprintf("%s/api/proxy/", node.URL)

			// 创建请求
			httpReq, err := http.NewRequest("GET", apiURL, nil)
			if err != nil {
				h.logger.Error("Failed to create request", "error", err, "url", apiURL)

				// 为该节点的所有隧道设置离线状态
				mu.Lock()
				for _, proxyID := range proxyIDs {
					results[strconv.FormatInt(proxyID, 10)] = gin.H{
						"status":          "offline",
						"proxyId":         proxyID,
						"proxyName":       proxyNameMap[proxyID],
						"nodeId":          nodeID,
						"nodeName":        nodeName,
						"lastStartTime":   "",
						"lastCloseTime":   "",
						"curConns":        0,
						"todayTrafficIn":  0,
						"todayTrafficOut": 0,
					}
				}
				mu.Unlock()
				return
			}

			// 设置Basic认证
			httpReq.SetBasicAuth(node.User, node.Token)

			// 发送请求
			resp, err := client.Do(httpReq)
			if err != nil {
				h.logger.Error("Failed to get proxy status", "error", err, "nodeID", nodeID)

				// 为该节点的所有隧道设置离线状态
				mu.Lock()
				for _, proxyID := range proxyIDs {
					results[strconv.FormatInt(proxyID, 10)] = gin.H{
						"status":          "offline",
						"proxyId":         proxyID,
						"proxyName":       proxyNameMap[proxyID],
						"nodeId":          nodeID,
						"nodeName":        nodeName,
						"lastStartTime":   "",
						"lastCloseTime":   "",
						"curConns":        0,
						"todayTrafficIn":  0,
						"todayTrafficOut": 0,
					}
				}
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			// 读取响应
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				h.logger.Error("Failed to read response", "error", err, "nodeID", nodeID)

				// 为该节点的所有隧道设置离线状态
				mu.Lock()
				for _, proxyID := range proxyIDs {
					results[strconv.FormatInt(proxyID, 10)] = gin.H{
						"status":          "offline",
						"proxyId":         proxyID,
						"proxyName":       proxyNameMap[proxyID],
						"nodeId":          nodeID,
						"nodeName":        nodeName,
						"lastStartTime":   "",
						"lastCloseTime":   "",
						"curConns":        0,
						"todayTrafficIn":  0,
						"todayTrafficOut": 0,
					}
				}
				mu.Unlock()
				return
			}

			// 解析响应
			var allProxiesStatus map[string]interface{}
			if err := json.Unmarshal(body, &allProxiesStatus); err != nil {
				h.logger.Error("Failed to parse response", "error", err, "body", string(body), "nodeID", nodeID)

				// 为该节点的所有隧道设置离线状态
				mu.Lock()
				for _, proxyID := range proxyIDs {
					results[strconv.FormatInt(proxyID, 10)] = gin.H{
						"status":          "offline",
						"proxyId":         proxyID,
						"proxyName":       proxyNameMap[proxyID],
						"nodeId":          nodeID,
						"nodeName":        nodeName,
						"lastStartTime":   "",
						"lastCloseTime":   "",
						"curConns":        0,
						"todayTrafficIn":  0,
						"todayTrafficOut": 0,
					}
				}
				mu.Unlock()
				return
			}

			// 筛选请求的隧道状态
			mu.Lock()
			defer mu.Unlock()

			// 收集所有类型的隧道
			allProxies := make([]interface{}, 0)
			proxyTypes := []string{"tcp", "udp", "http", "https", "stcp", "xtcp"}

			for _, pType := range proxyTypes {
				if typeProxiesList, ok := allProxiesStatus[pType].(map[string]interface{}); ok {
					if proxiesList, ok := typeProxiesList["proxies"].([]interface{}); ok {
						allProxies = append(allProxies, proxiesList...)
					}
				}
			}

			// 将隧道列表转换为map以便快速查找
			proxyStatusMap := make(map[string]interface{})
			for _, proxyData := range allProxies {
				proxyInfo, ok := proxyData.(map[string]interface{})
				if !ok {
					continue
				}

				name, ok := proxyInfo["name"].(string)
				if !ok {
					continue
				}

				proxyStatusMap[name] = proxyInfo
			}

			for _, proxyID := range proxyIDs {
				proxyName := proxyNameMap[proxyID]
				expectedProxyName := user.Username + "." + proxyName

				// 在节点返回的所有代理中查找该代理
				proxyStatus, found := proxyStatusMap[expectedProxyName]
				if found {
					// 解析并提取所需的特定状态信息
					proxyInfo, ok := proxyStatus.(map[string]interface{})
					if !ok {
						proxyInfo = make(map[string]interface{})
					}

					// 获取状态信息，默认值为空或0
					status := "offline"
					if statusStr, ok := proxyInfo["status"].(string); ok && statusStr == "online" {
						status = "online"
					}

					lastStartTime := ""
					if start, ok := proxyInfo["lastStartTime"].(string); ok {
						lastStartTime = start
					}

					lastCloseTime := ""
					if close, ok := proxyInfo["lastCloseTime"].(string); ok {
						lastCloseTime = close
					}

					curConns := 0
					if conns, ok := proxyInfo["curConns"].(float64); ok {
						curConns = int(conns)
					}

					trafficIn := int64(0)
					if in, ok := proxyInfo["todayTrafficIn"].(float64); ok {
						trafficIn = int64(in)
					}

					trafficOut := int64(0)
					if out, ok := proxyInfo["todayTrafficOut"].(float64); ok {
						trafficOut = int64(out)
					}

					results[strconv.FormatInt(proxyID, 10)] = gin.H{
						"status":          status,
						"proxyId":         proxyID,
						"proxyName":       proxyName,
						"nodeId":          nodeID,
						"nodeName":        nodeName,
						"lastStartTime":   lastStartTime,
						"lastCloseTime":   lastCloseTime,
						"curConns":        curConns,
						"todayTrafficIn":  trafficIn,
						"todayTrafficOut": trafficOut,
					}
				} else {
					// 未找到代理状态，设置为离线状态
					results[strconv.FormatInt(proxyID, 10)] = gin.H{
						"status":          "offline",
						"proxyId":         proxyID,
						"proxyName":       proxyName,
						"nodeId":          nodeID,
						"nodeName":        nodeName,
						"lastStartTime":   "",
						"lastCloseTime":   "",
						"curConns":        0,
						"todayTrafficIn":  0,
						"todayTrafficOut": 0,
					}
				}
			}
		}(nodeID, proxyIDs)
	}

	// 等待所有goroutine完成
	wg.Wait()

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取成功",
		"data": results,
	})
}
