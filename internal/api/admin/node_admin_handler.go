package admin

import (
	"context"
	"database/sql"
	"net/http"
	"stellarfrp/internal/repository"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// NodeAdminHandler 节点管理处理器
type NodeAdminHandler struct {
	nodeService service.NodeService
	nodeRepo    repository.NodeRepository
	logger      *logger.Logger
}

// NewNodeAdminHandler 创建节点管理处理器实例
func NewNodeAdminHandler(nodeService service.NodeService, nodeRepo repository.NodeRepository, logger *logger.Logger) *NodeAdminHandler {
	return &NodeAdminHandler{
		nodeService: nodeService,
		nodeRepo:    nodeRepo,
		logger:      logger,
	}
}

// ListNodes 获取节点列表
func (h *NodeAdminHandler) ListNodes(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无效的页码"})
		return
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无效的每页数量"})
		return
	}

	offset := (page - 1) * pageSize

	// 获取节点列表
	nodes, err := h.nodeService.List(context.Background(), offset, pageSize)
	if err != nil {
		h.logger.Error("获取节点列表失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点列表失败"})
		return
	}

	// 获取所有节点数量（用于计算总页数）
	allNodes, err := h.nodeService.GetAllNodes(context.Background())
	if err != nil {
		h.logger.Error("获取节点总数失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点总数失败"})
		return
	}

	total := int64(len(allNodes))
	pages := (total + int64(pageSize) - 1) / int64(pageSize)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取成功",
		"pagination": gin.H{
			"page":      page,
			"page_size": pageSize,
			"pages":     pages,
			"total":     total,
		},
		"nodes": nodes,
	})
}

// GetNode 获取单个节点信息
func (h *NodeAdminHandler) GetNode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无效的节点ID"})
		return
	}

	node, err := h.nodeService.GetByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "节点不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取成功",
		"data": node,
	})
}

// CreateNodeRequest 创建节点请求
type CreateNodeRequest struct {
	NodeName     string  `json:"node_name" binding:"required"`
	FrpsPort     int     `json:"frps_port" binding:"required"`
	URL          string  `json:"url" binding:"required"`
	Token        string  `json:"token" binding:"required"`
	User         string  `json:"user" binding:"required"`
	Description  *string `json:"description"`
	Permission   string  `json:"permission" binding:"required"`    // JSON格式的字符串，如["1","2"]
	AllowedTypes string  `json:"allowed_types" binding:"required"` // JSON格式的字符串，如["TCP","UDP"]
	Host         *string `json:"host"`
	PortRange    string  `json:"port_range" binding:"required"`
	IP           string  `json:"ip" binding:"required"`
	Status       int     `json:"status" binding:"required"`
}

// CreateNode 创建节点
func (h *NodeAdminHandler) CreateNode(c *gin.Context) {
	var req CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("创建节点参数绑定失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误：" + err.Error()})
		return
	}

	// 检查节点名是否已存在
	existingNode, err := h.nodeService.GetByNodeName(context.Background(), req.NodeName)
	if err == nil && existingNode != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "节点名已存在"})
		return
	}

	// 创建节点对象
	var description sql.NullString
	if req.Description != nil {
		description = sql.NullString{String: *req.Description, Valid: true}
	}

	var host sql.NullString
	if req.Host != nil {
		host = sql.NullString{String: *req.Host, Valid: true}
	}

	node := &repository.Node{
		NodeName:     req.NodeName,
		FrpsPort:     req.FrpsPort,
		URL:          req.URL,
		Token:        req.Token,
		User:         req.User,
		Description:  description,
		Permission:   req.Permission,
		AllowedTypes: req.AllowedTypes,
		Host:         host,
		PortRange:    req.PortRange,
		IP:           req.IP,
		Status:       req.Status,
	}

	// 保存节点
	err = h.nodeRepo.Create(context.Background(), node)
	if err != nil {
		h.logger.Error("创建节点失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "创建节点失败：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "创建成功",
		"data": node,
	})
}

// UpdateNodeRequest 更新节点请求
type UpdateNodeRequest struct {
	NodeName     *string `json:"node_name"`
	FrpsPort     *int    `json:"frps_port"`
	URL          *string `json:"url"`
	Token        *string `json:"token"`
	User         *string `json:"user"`
	Description  *string `json:"description"`
	Permission   *string `json:"permission"`
	AllowedTypes *string `json:"allowed_types"`
	Host         *string `json:"host"`
	PortRange    *string `json:"port_range"`
	IP           *string `json:"ip"`
	Status       *int    `json:"status"`
	ID           *int64  `json:"id"`
}

// UpdateNode 更新节点
func (h *NodeAdminHandler) UpdateNode(c *gin.Context) {
	// 解析请求体
	var req UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("更新节点参数绑定失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误：" + err.Error()})
		return
	}

	if req.ID == nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "节点ID不能为空"})
		return
	}

	id := *req.ID

	// 获取现有节点
	node, err := h.nodeService.GetByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "节点不存在"})
		return
	}

	// 更新节点信息
	if req.NodeName != nil {
		// 检查新节点名是否已存在（如果更改了节点名）
		if *req.NodeName != node.NodeName {
			existingNode, err := h.nodeService.GetByNodeName(context.Background(), *req.NodeName)
			if err == nil && existingNode != nil {
				c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "节点名已存在"})
				return
			}
		}
		node.NodeName = *req.NodeName
	}
	if req.FrpsPort != nil {
		node.FrpsPort = *req.FrpsPort
	}
	if req.URL != nil {
		node.URL = *req.URL
	}
	if req.Token != nil {
		node.Token = *req.Token
	}
	if req.User != nil {
		node.User = *req.User
	}
	if req.Description != nil {
		node.Description = sql.NullString{String: *req.Description, Valid: true}
	}
	if req.Permission != nil {
		node.Permission = *req.Permission
	}
	if req.AllowedTypes != nil {
		node.AllowedTypes = *req.AllowedTypes
	}
	if req.Host != nil {
		node.Host = sql.NullString{String: *req.Host, Valid: true}
	}
	if req.PortRange != nil {
		node.PortRange = *req.PortRange
	}
	if req.IP != nil {
		node.IP = *req.IP
	}
	if req.Status != nil {
		node.Status = *req.Status
	}

	// 保存更新
	err = h.nodeRepo.Update(context.Background(), node)
	if err != nil {
		h.logger.Error("更新节点失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "更新节点失败：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "更新成功",
		"data": node,
	})
}

// DeleteNodeRequest 删除节点请求
type DeleteNodeRequest struct {
	ID int64 `json:"id" binding:"required"`
}

// DeleteNode 删除节点
func (h *NodeAdminHandler) DeleteNode(c *gin.Context) {
	var req DeleteNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("删除节点参数绑定失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误：" + err.Error()})
		return
	}

	id := req.ID

	// 检查节点是否存在
	_, err := h.nodeService.GetByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "节点不存在"})
		return
	}

	// 删除节点
	err = h.nodeRepo.Delete(context.Background(), id)
	if err != nil {
		h.logger.Error("删除节点失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "删除节点失败：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "删除成功",
	})
}

// SearchNodes 搜索节点
func (h *NodeAdminHandler) SearchNodes(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "搜索关键词不能为空"})
		return
	}

	// 获取所有节点
	allNodes, err := h.nodeService.GetAllNodes(context.Background())
	if err != nil {
		h.logger.Error("获取节点列表失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "搜索节点失败"})
		return
	}

	// 过滤符合搜索条件的节点
	var filteredNodes []*repository.Node
	for _, node := range allNodes {
		// 根据节点名称或描述进行模糊匹配
		if containsIgnoreCase(node.NodeName, keyword) ||
			(node.Description.Valid && containsIgnoreCase(node.Description.String, keyword)) {
			filteredNodes = append(filteredNodes, node)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":  200,
		"msg":   "搜索成功",
		"nodes": filteredNodes,
	})
}

// containsIgnoreCase 判断字符串是否包含子串（忽略大小写）
func containsIgnoreCase(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	return strings.Contains(s, substr)
}
