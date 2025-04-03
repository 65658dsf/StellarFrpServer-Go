package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stellarfrp/internal/repository"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"
	"strconv"
	"sync"
	"time"

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

	// 构建返回数据 - 使用对象格式，节点ID作为键
	nodeMap := make(map[string]gin.H)
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
		if node.Description.Valid {
			if err := json.Unmarshal([]byte(node.Description.String), &description); err != nil {
				// 如果解析失败，使用原始字符串
				description = node.Description.String
			}
		} else {
			description = ""
		}

		// 使用节点ID作为键
		nodeID := strconv.FormatInt(node.ID, 10)
		nodeMap[nodeID] = gin.H{
			"NodeName":     node.NodeName,
			"AllowedTypes": allowedTypes,
			"PortRange":    node.PortRange,
			"Status":       node.Status,
			"Description":  description,
			"ID":           nodeID,
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "获取成功", "data": nodeMap})
}

// NodeInfoResponse 节点服务器信息响应
type NodeInfoResponse struct {
	Version         string         `json:"version"`
	BindPort        int            `json:"bindPort"`
	TotalTrafficIn  int64          `json:"totalTrafficIn"`
	TotalTrafficOut int64          `json:"totalTrafficOut"`
	CurConns        int            `json:"curConns"`
	ClientCounts    int            `json:"clientCounts"`
	ProxyTypeCount  map[string]int `json:"proxyTypeCount"`
}

// 格式化流量大小为带单位的字符串
func formatTraffic(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	if bytes < KB {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < MB {
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	} else if bytes < GB {
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	} else if bytes < TB {
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	}
	return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
}

// GetNodesInfo 获取所有节点的信息
func (h *NodeHandler) GetNodesInfo(c *gin.Context) {
	// 从请求头获取token
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未授权，请先登录"})
		return
	}

	// 通过token验证用户是否存在，但不需要使用用户信息
	_, err := h.userService.GetByToken(context.Background(), token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的token"})
		return
	}

	// 获取所有节点，不再根据用户权限过滤
	nodes, err := h.nodeService.GetAllNodes(context.Background())
	if err != nil {
		h.logger.Error("Failed to get all nodes", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点列表失败"})
		return
	}

	// 创建HTTP客户端，设置超时
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 并发获取每个节点的信息
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(map[string]gin.H)

	for _, node := range nodes {
		wg.Add(1)
		go func(node *repository.Node) {
			defer wg.Done()

			// 构建节点API URL
			apiURL := fmt.Sprintf("%s/api/serverinfo", node.URL)

			// 创建请求
			req, err := http.NewRequest("GET", apiURL, nil)
			if err != nil {
				h.logger.Error("Failed to create request", "error", err, "url", apiURL)
				mu.Lock()
				results[strconv.FormatInt(node.ID, 10)] = gin.H{
					"NodeName":   node.NodeName,
					"Status":     "offline",
					"Error":      "请求创建失败",
					"Version":    "",
					"Clients":    0,
					"TrafficIn":  "",
					"TrafficOut": "",
				}
				mu.Unlock()
				return
			}

			// 设置Basic认证
			req.SetBasicAuth(node.User, node.Token)

			// 发送请求
			resp, err := client.Do(req)
			if err != nil {
				h.logger.Error("Failed to get node info", "error", err, "node", node.NodeName)
				mu.Lock()
				results[strconv.FormatInt(node.ID, 10)] = gin.H{
					"NodeName":   node.NodeName,
					"Status":     "offline",
					"Error":      err.Error(),
					"Version":    "",
					"Clients":    0,
					"TrafficIn":  "",
					"TrafficOut": "",
				}
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			// 读取响应
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				h.logger.Error("Failed to read response", "error", err, "node", node.NodeName)
				mu.Lock()
				results[strconv.FormatInt(node.ID, 10)] = gin.H{
					"NodeName":   node.NodeName,
					"Status":     "offline",
					"Error":      "响应读取失败",
					"Version":    "",
					"Clients":    0,
					"TrafficIn":  "",
					"TrafficOut": "",
				}
				mu.Unlock()
				return
			}

			// 解析JSON响应
			var nodeInfo NodeInfoResponse
			if err := json.Unmarshal(body, &nodeInfo); err != nil {
				h.logger.Error("Failed to parse response", "error", err, "node", node.NodeName)
				mu.Lock()
				results[strconv.FormatInt(node.ID, 10)] = gin.H{
					"NodeName":   node.NodeName,
					"Status":     "offline",
					"Error":      "响应解析失败",
					"Version":    "",
					"Clients":    0,
					"TrafficIn":  "",
					"TrafficOut": "",
				}
				mu.Unlock()
				return
			}

			// 格式化流量数据
			trafficIn := formatTraffic(nodeInfo.TotalTrafficIn)
			trafficOut := formatTraffic(nodeInfo.TotalTrafficOut)

			// 保存结果
			mu.Lock()
			results[strconv.FormatInt(node.ID, 10)] = gin.H{
				"NodeName":   node.NodeName,
				"Status":     "online",
				"Version":    nodeInfo.Version,
				"Clients":    nodeInfo.ClientCounts,
				"TrafficIn":  trafficIn,
				"TrafficOut": trafficOut,
			}
			mu.Unlock()
		}(node)
	}

	// 等待所有goroutine完成
	wg.Wait()

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取成功",
		"data": results,
	})
}
