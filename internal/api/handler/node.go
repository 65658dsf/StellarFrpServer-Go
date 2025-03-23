package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"

	"github.com/gin-gonic/gin"
)

// NodeHandler 节点处理器
type NodeHandler struct {
	nodeService service.NodeService
	userService service.UserService
	logger      *logger.Logger
}

// NewNodeHandler 创建节点处理器实例
func NewNodeHandler(nodeService service.NodeService, userService service.UserService, logger *logger.Logger) *NodeHandler {
	return &NodeHandler{
		nodeService: nodeService,
		userService: userService,
		logger:      logger,
	}
}

// GetAccessibleNodes 获取用户可访问的节点列表
func (h *NodeHandler) GetAccessibleNodes(c *gin.Context) {
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

	// 获取用户组可访问的节点列表
	nodes, err := h.nodeService.GetAccessibleNodes(context.Background(), user.GroupID)
	if err != nil {
		h.logger.Error("Failed to get accessible nodes", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	// 构建返回数据
	nodeList := make([]gin.H, 0, len(nodes))
	for _, node := range nodes {
		// 解析AllowedTypes字段，它是JSON格式的字符串
		var allowedTypes []string
		if err := json.Unmarshal([]byte(node.AllowedTypes), &allowedTypes); err != nil {
			// 如果解析失败，使用原始字符串
			h.logger.Error("Failed to parse allowed_types", "error", err, "value", node.AllowedTypes)
			allowedTypes = []string{node.AllowedTypes}
		}

		// 解析Description字段，如果它也是JSON格式的字符串
		var description interface{}
		if err := json.Unmarshal([]byte(node.Description), &description); err != nil {
			// 如果解析失败，使用原始字符串
			description = node.Description
		}

		nodeList = append(nodeList, gin.H{
			"node_name":     node.NodeName,
			"allowed_types": allowedTypes,
			"port_range":    node.PortRange,
			"status":        node.Status,
			"description":   description,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "获取成功", "data": nodeList})
}
