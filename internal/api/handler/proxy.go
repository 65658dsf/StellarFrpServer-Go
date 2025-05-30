package handler

import (
	"bytes"
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
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未授权，请先登录"})
		return
	}

	user, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的token"})
		return
	}

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

	node, err := h.nodeService.GetByID(context.Background(), req.NodeID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "节点不存在或已下线"})
		return
	}

	nodes, err := h.nodeService.GetAccessibleNodes(context.Background(), user.GroupID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点权限失败"})
		return
	}

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

	if req.ProxyType != "http" && req.ProxyType != "https" && req.RemotePort != 0 {
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

	if req.ProxyType == "http" || req.ProxyType == "https" {
		if req.Domain == "" {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "HTTP/HTTPS类型的隧道必须填写域名"})
			return
		}

		expectedPort := 80
		if req.ProxyType == "https" {
			expectedPort = 443
		}
		if req.RemotePort != expectedPort {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": req.ProxyType + "类型的隧道远程端口必须为" + strconv.Itoa(expectedPort)})
			return
		}
	} else if req.ProxyType == "tcp" || req.ProxyType == "udp" {
		if req.RemotePort == 0 {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "TCP/UDP类型的隧道必须填写远程端口"})
			return
		}

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

	proxyCount, err := h.proxyService.GetUserProxyCount(context.Background(), user.Username)
	if err != nil {
		h.logger.Error("Failed to get user proxy count", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取隧道数量失败"})
		return
	}

	tunnelLimit, err := h.userService.GetGroupTunnelLimit(context.Background(), user.GroupID)
	if err != nil {
		h.logger.Error("Failed to get group tunnel limit", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取用户组隧道限制失败"})
		return
	}

	additionalTunnels := 0
	if user.TunnelCount != nil {
		additionalTunnels = *user.TunnelCount
	}

	totalTunnelLimit := tunnelLimit + additionalTunnels

	if proxyCount >= totalTunnelLimit {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "您已达到可创建的数量上限"})
		return
	}

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
		Status:            "offline",
	}

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
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未授权，请先登录"})
		return
	}

	user, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的token"})
		return
	}

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

	if existingProxy.Username != user.Username {
		c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "您没有权限修改此隧道"})
		return
	}

	node, err := h.nodeService.GetByID(context.Background(), req.NodeID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "节点不存在或已下线"})
		return
	}

	if existingProxy.Node != req.NodeID {
		originalNode, err := h.nodeService.GetByID(context.Background(), existingProxy.Node)
		if err == nil {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "该隧道属于 " + originalNode.NodeName + " 节点，不能修改为其他节点"})
		} else {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "该隧道不属于请求中指定的节点"})
		}
		return
	}

	nodes, err := h.nodeService.GetAccessibleNodes(context.Background(), user.GroupID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点权限失败"})
		return
	}

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

	if req.ProxyType != "http" && req.ProxyType != "https" && req.RemotePort != 0 {
		isUsed, err := h.proxyService.IsRemotePortUsed(context.Background(), req.NodeID, req.ProxyType, strconv.Itoa(req.RemotePort))
		if err != nil {
			h.logger.Error("Failed to check remote port usage", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "检查端口占用失败"})
			return
		}
		if isUsed && (existingProxy.RemotePort != strconv.Itoa(req.RemotePort) || existingProxy.Node != req.NodeID) {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "该节点下已有相同协议类型的隧道使用了端口 " + strconv.Itoa(req.RemotePort) + "，请更换端口"})
			return
		}
	}

	if req.ProxyType == "http" || req.ProxyType == "https" {
		if req.Domain == "" {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "HTTP/HTTPS类型的隧道必须填写域名"})
			return
		}

		expectedPort := 80
		if req.ProxyType == "https" {
			expectedPort = 443
		}
		if req.RemotePort != expectedPort {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": req.ProxyType + "类型的隧道远程端口必须为" + strconv.Itoa(expectedPort)})
			return
		}
	} else if req.ProxyType == "tcp" || req.ProxyType == "udp" {
		if req.RemotePort == 0 {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "TCP/UDP类型的隧道必须填写远程端口"})
			return
		}

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
		Status:            existingProxy.Status,
	}

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
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未授权，请先登录"})
		return
	}

	user, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的token"})
		return
	}

	type DeleteRequest struct {
		ID int64 `json:"id" binding:"required"`
	}

	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

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

	if proxy.Username != user.Username {
		c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "您没有权限删除此隧道"})
		return
	}

	err = h.proxyService.Delete(context.Background(), req.ID)
	if err != nil {
		h.logger.Error("Failed to delete proxy", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "删除隧道失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "删除成功"})
}

// generateProxyConfigString 生成隧道配置字符串的辅助函数
func (h *ProxyHandler) generateProxyConfigString(proxy *repository.Proxy, node *repository.Node, userToken string, bandwidthStr string) string {
	var configBuilder strings.Builder

	// 基本配置信息
	baseConfig := fmt.Sprintf("serverAddr = \"%s\"\nserverPort = %d\nuser = \"%s\"\nmetadatas.token = \"%s\"\n\n",
		node.IP, node.FrpsPort, proxy.Username, userToken)
	configBuilder.WriteString(baseConfig)

	// 隧道类型特定配置
	typeConfig := fmt.Sprintf("[[proxies]]\nname = \"%s\"\ntype = \"%s\"\n", proxy.ProxyName, proxy.ProxyType)
	configBuilder.WriteString(typeConfig)

	switch proxy.ProxyType {
	case "http":
		configBuilder.WriteString(fmt.Sprintf("localIP = \"%s\"\nlocalPort = %d\n", proxy.LocalIP, proxy.LocalPort))
		if proxy.Domain != "" {
			configBuilder.WriteString(fmt.Sprintf("customDomains = [\"%s\"]\n", proxy.Domain))
		}
	case "https":
		if proxy.Domain != "" {
			configBuilder.WriteString(fmt.Sprintf("customDomains = [\"%s\"]\n", proxy.Domain))
		}
		configBuilder.WriteString(fmt.Sprintf("[proxies.plugin]\ntype = \"https2http\"\nlocalAddr = \"%s:%d\"\n# HTTPS 证书相关的配置\ncrtPath = \"./server.crt\"\nkeyPath = \"./server.key\"\n", proxy.LocalIP, proxy.LocalPort))
		if proxy.HostHeaderRewrite != "" {
			configBuilder.WriteString(fmt.Sprintf("hostHeaderRewrite = \"%s\"\n", proxy.HostHeaderRewrite))
		}
		if proxy.HeaderXFromWhere != "" {
			configBuilder.WriteString(fmt.Sprintf("requestHeaders.set.x-from-where = \"%s\"\n", proxy.HeaderXFromWhere))
		}
	default: // tcp, udp 和其他类型
		configBuilder.WriteString(fmt.Sprintf("localIP = \"%s\"\nlocalPort = %d\nremotePort = %s\n", proxy.LocalIP, proxy.LocalPort, proxy.RemotePort))
	}

	// 通用传输配置
	transportConfig := fmt.Sprintf("\n[proxies.transport]\nuseEncryption = %s\nuseCompression = %s\nbandwidthLimit = %s\nbandwidthLimitMode = \"server\"\n",
		proxy.UseEncryption, proxy.UseCompression, bandwidthStr)
	configBuilder.WriteString(transportConfig)

	return configBuilder.String()
}

// GetProxyByID 根据ID获取隧道
func (h *ProxyHandler) GetProxyByID(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未授权，请先登录"})
		return
	}

	user, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的token"})
		return
	}

	type GetProxyRequest struct {
		ID       int64 `json:"id"`
		Page     int   `json:"page"`
		PageSize int   `json:"page_size"`
	}

	var req GetProxyRequest
	jsonBindErr := c.ShouldBindJSON(&req)
	if jsonBindErr != nil && jsonBindErr.Error() != "EOF" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误: " + jsonBindErr.Error()})
		return
	}

	userRequestedPage := 0
	userRequestedPageSize := 0

	pageQueryStr := c.Query("page")
	pageSizeQueryStr := c.Query("page_size")

	if pVal, err := strconv.Atoi(pageQueryStr); err == nil && pVal > 0 {
		userRequestedPage = pVal
	} else if req.Page > 0 {
		userRequestedPage = req.Page
	}

	if psVal, err := strconv.Atoi(pageSizeQueryStr); err == nil && psVal > 0 {
		userRequestedPageSize = psVal
	} else if req.PageSize > 0 {
		userRequestedPageSize = req.PageSize
	}

	userGroup, err := h.userService.GetUserGroup(context.Background(), user.ID)
	if err != nil {
		h.logger.Error("Failed to get user group", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取用户组失败"})
		return
	}

	userBandwidth := 0
	if user.Bandwidth != nil {
		userBandwidth = *user.Bandwidth
	}

	totalBandwidth := userGroup.BandwidthLimit + userBandwidth
	bandwidthStr := fmt.Sprintf("\"%dMB\"", totalBandwidth)

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

		if proxy.Username != user.Username {
			c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "您没有权限查看此隧道"})
			return
		}

		node, err := h.nodeService.GetByID(context.Background(), proxy.Node)
		if err != nil {
			h.logger.Error("Failed to get node", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点信息失败"})
			return
		}

		remotePort, _ := strconv.Atoi(proxy.RemotePort)
		link := ""
		if proxy.ProxyType != "http" && proxy.ProxyType != "https" {
			if node.Host.Valid && node.Host.String != "" {
				link = node.Host.String + ":" + proxy.RemotePort
			} else {
				link = node.IP + ":" + proxy.RemotePort
			}
		} else {
			link = proxy.Domain
		}

		data := h.generateProxyConfigString(proxy, node, user.Token, bandwidthStr)

		tunnelData := gin.H{
			"Id":         proxy.ID,
			"NodeId":     proxy.Node,
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

	var proxies []*repository.Proxy
	var totalCount int

	responsePage := 1
	responsePageSize := 10

	shouldPaginate := userRequestedPage > 0 && userRequestedPageSize > 0

	if shouldPaginate {
		responsePage = userRequestedPage
		responsePageSize = userRequestedPageSize

		var countErr error
		totalCount, countErr = h.proxyService.GetUserProxyCount(context.Background(), user.Username)
		if countErr != nil {
			h.logger.Error("Failed to get total count of proxies", "error", countErr)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取隧道总数失败: " + countErr.Error()})
			return
		}

		offset := (responsePage - 1) * responsePageSize
		proxies, err = h.proxyService.GetByUsernameWithPagination(context.Background(), user.Username, offset, responsePageSize)
	} else {
		proxies, err = h.proxyService.GetByUsername(context.Background(), user.Username)
		if err == nil {
			totalCount = len(proxies)
		}
		responsePage = 1
		if totalCount > 0 {
			responsePageSize = totalCount
		} else {
			responsePageSize = 10
		}
	}

	if err != nil {
		h.logger.Error("Failed to get proxies", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取隧道列表失败: " + err.Error()})
		return
	}

	tunnels := make(gin.H)

	for _, proxy := range proxies {
		node, err := h.nodeService.GetByID(context.Background(), proxy.Node)
		if err != nil {
			h.logger.Warn("Failed to get node info for proxy in list", "proxyID", proxy.ID, "nodeID", proxy.Node, "error", err)
			continue
		}

		remotePort, _ := strconv.Atoi(proxy.RemotePort)
		link := ""
		if proxy.ProxyType != "http" && proxy.ProxyType != "https" {
			if node.Host.Valid && node.Host.String != "" {
				link = node.Host.String + ":" + proxy.RemotePort
			} else {
				link = node.IP + ":" + proxy.RemotePort
			}
		} else {
			link = proxy.Domain
		}

		data := h.generateProxyConfigString(proxy, node, user.Token, bandwidthStr)

		tunnels[strconv.FormatInt(proxy.ID, 10)] = gin.H{
			"Id":         proxy.ID,
			"NodeId":     proxy.Node,
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

	pages := 0
	if responsePageSize > 0 && totalCount > 0 {
		pages = (totalCount + responsePageSize - 1) / responsePageSize
	} else if totalCount == 0 {
		pages = 0
	} else {
		pages = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"msg":    "获取成功",
		"tunnel": tunnels,
		"pagination": gin.H{
			"total":     totalCount,
			"page":      responsePage,
			"page_size": responsePageSize,
			"pages":     pages,
		},
	})
}

// GetProxyStatus 获取隧道状态
func (h *ProxyHandler) GetProxyStatus(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未授权，请先登录"})
		return
	}

	user, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的token"})
		return
	}

	type StatusRequest struct {
		IDs []string `json:"id"`
	}

	var reqData StatusRequest
	if err := c.ShouldBindJSON(&reqData); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	if len(reqData.IDs) == 0 {
		proxies, err := h.proxyService.GetByUsername(context.Background(), user.Username)
		if err != nil {
			h.logger.Error("Failed to get user's proxies", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取用户隧道列表失败"})
			return
		}

		if len(proxies) == 0 {
			c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "获取成功", "data": gin.H{}})
			return
		}

		reqData.IDs = make([]string, len(proxies))
		for i, proxy := range proxies {
			reqData.IDs[i] = strconv.FormatInt(proxy.ID, 10)
		}
	}

	proxyNodeMap := make(map[int64]int64)
	proxyNameMap := make(map[int64]string)
	nodeNameMap := make(map[int64]string)
	nodeMap := make(map[int64]*repository.Node)

	for _, idStr := range reqData.IDs {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			h.logger.Error("Invalid proxy ID in status request", "id", idStr, "error", err)
			continue
		}

		proxy, err := h.proxyService.GetByID(context.Background(), id)
		if err != nil {
			h.logger.Error("Failed to get proxy for status", "id", id, "error", err)
			continue
		}

		if proxy == nil {
			continue
		}

		if proxy.Username != user.Username {
			continue
		}

		node, err := h.nodeService.GetByID(context.Background(), proxy.Node)
		if err != nil {
			h.logger.Error("Failed to get node for status", "nodeID", proxy.Node, "error", err)
			continue
		}

		nodeNameMap[proxy.Node] = node.NodeName
		nodeMap[proxy.Node] = node

		proxyNodeMap[proxy.ID] = proxy.Node
		proxyNameMap[proxy.ID] = proxy.ProxyName
	}

	if len(proxyNodeMap) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "未找到有效的隧道"})
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	var mu sync.Mutex
	results := make(map[string]gin.H)

	nodeProxiesMap := make(map[int64][]int64)

	for proxyID, nodeID := range proxyNodeMap {
		nodeProxiesMap[nodeID] = append(nodeProxiesMap[nodeID], proxyID)
	}

	var wg sync.WaitGroup

	for nodeID, proxyIDs := range nodeProxiesMap {
		wg.Add(1)
		go func(nodeID int64, proxyIDs []int64) {
			defer wg.Done()

			node := nodeMap[nodeID]
			nodeName := nodeNameMap[nodeID]

			apiURL := fmt.Sprintf("%s/api/proxy/", node.URL)

			httpReq, err := http.NewRequest("GET", apiURL, nil)
			if err != nil {
				h.logger.Error("Failed to create request for node status", "error", err, "url", apiURL)
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

			httpReq.SetBasicAuth(node.User, node.Token)

			resp, err := client.Do(httpReq)
			if err != nil {
				h.logger.Error("Failed to get proxy status from node", "error", err, "nodeID", nodeID)
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

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				h.logger.Error("Failed to read response from node status", "error", err, "nodeID", nodeID)
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

			var allProxiesStatus map[string]interface{}
			if err := json.Unmarshal(body, &allProxiesStatus); err != nil {
				h.logger.Error("Failed to parse response from node status", "error", err, "body", string(body), "nodeID", nodeID)
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

			mu.Lock()
			defer mu.Unlock()

			allProxies := make([]interface{}, 0)
			proxyTypes := []string{"tcp", "udp", "http", "https", "stcp", "xtcp"}

			for _, pType := range proxyTypes {
				if typeProxiesList, ok := allProxiesStatus[pType].(map[string]interface{}); ok {
					if proxiesList, ok := typeProxiesList["proxies"].([]interface{}); ok {
						allProxies = append(allProxies, proxiesList...)
					}
				}
			}

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

				proxyStatus, found := proxyStatusMap[expectedProxyName]
				if found {
					proxyInfo, ok := proxyStatus.(map[string]interface{})
					if !ok {
						proxyInfo = make(map[string]interface{})
					}

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

	wg.Wait()

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取成功",
		"data": results,
	})
}

// CloseProxy 关闭隧道
func (h *ProxyHandler) CloseProxy(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未授权，请先登录"})
		return
	}

	user, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的token"})
		return
	}

	type CloseRequest struct {
		ID int64 `json:"id" binding:"required"`
	}

	var req CloseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	proxy, err := h.proxyService.GetByID(context.Background(), req.ID)
	if err != nil {
		h.logger.Error("获取隧道信息失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取隧道信息失败"})
		return
	}

	if proxy == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "隧道不存在"})
		return
	}

	if proxy.Username != user.Username {
		c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "您没有权限操作该隧道"})
		return
	}

	if proxy.RunID == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "该隧道未运行"})
		return
	}

	node, err := h.nodeService.GetByID(context.Background(), proxy.Node)
	if err != nil {
		h.logger.Error("获取节点信息失败", "error", err, "nodeID", proxy.Node)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点信息失败"})
		return
	}

	requestBody, err := json.Marshal(map[string]string{
		"runid": proxy.RunID,
	})
	if err != nil {
		h.logger.Error("构建请求体失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	apiURL := fmt.Sprintf("%s/api/client/kick", node.URL)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req2, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		h.logger.Error("创建请求失败", "error", err, "url", apiURL)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	req2.Header.Set("Content-Type", "application/json")
	req2.SetBasicAuth(node.User, node.Token)

	resp, err := client.Do(req2)
	if err != nil {
		h.logger.Error("发送请求失败", "error", err, "url", apiURL)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "关闭隧道请求失败"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("读取响应失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "读取响应失败"})
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		h.logger.Error("解析响应失败", "error", err, "body", string(body))
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "解析响应失败"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("节点返回错误", "statusCode", resp.StatusCode, "response", string(body))
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "节点返回错误"})
		return
	}

	proxy.RunID = ""
	proxy.Status = "offline"
	if err := h.proxyService.Update(context.Background(), proxy); err != nil {
		h.logger.Error("更新隧道状态失败", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "隧道已成功关闭",
		"data": gin.H{
			"id":        proxy.ID,
			"proxyName": proxy.ProxyName,
		},
	})
}
