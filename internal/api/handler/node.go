package handler

import (
	"context"
	"database/sql"
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
	"github.com/redis/go-redis/v9"
)

// NodeHandler 节点处理器
type NodeHandler struct {
	nodeService service.NodeService
	userService service.UserService
	logger      *logger.Logger
	redisClient *redis.Client
}

// NewNodeHandler 创建节点处理器实例
func NewNodeHandler(nodeService service.NodeService, userService service.UserService, logger *logger.Logger, redisClient *redis.Client) *NodeHandler {
	return &NodeHandler{
		nodeService: nodeService,
		userService: userService,
		logger:      logger,
		redisClient: redisClient,
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
		// 跳过状态为2（待审核）的节点
		if node.Status == 2 {
			continue
		}

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

	// 获取每个节点的总流量数据
	nodeTrafficMap := make(map[string]*repository.NodeTrafficLog)
	ctx := context.Background()

	// 获取所有节点的总流量和
	totalTrafficIn, totalTrafficOut, err := h.nodeService.GetTotalTraffic(ctx)
	if err != nil {
		h.logger.Error("Failed to get total traffic", "error", err)
		// 错误不中断流程，继续处理
	}

	// 格式化总流量
	totalInFormatted := formatTraffic(totalTrafficIn)
	totalOutFormatted := formatTraffic(totalTrafficOut)

	for _, node := range nodes {
		// 获取最新的流量记录
		trafficLog, err := h.nodeService.GetLatestNodeTraffic(ctx, node.NodeName)
		if err != nil {
			h.logger.Error("Failed to get node traffic", "error", err, "node", node.NodeName)
			// 错误不中断流程，继续处理其他节点
			continue
		}

		if trafficLog != nil {
			nodeTrafficMap[node.NodeName] = trafficLog
		}
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
		// 跳过状态为2（待审核）的节点
		if node.Status == 2 {
			continue
		}

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
					"NodeName":        node.NodeName,
					"Status":          "offline",
					"Error":           "请求创建失败",
					"Version":         "",
					"Clients":         0,
					"TrafficIn":       "",
					"TrafficOut":      "",
					"TotalTrafficIn":  "",
					"TotalTrafficOut": "",
					"Load":            "0.00%",
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
					"NodeName":        node.NodeName,
					"Status":          "offline",
					"Error":           err.Error(),
					"Version":         "",
					"Clients":         0,
					"TrafficIn":       "",
					"TrafficOut":      "",
					"TotalTrafficIn":  "",
					"TotalTrafficOut": "",
					"Load":            "0.00%",
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
					"NodeName":        node.NodeName,
					"Status":          "offline",
					"Error":           "响应读取失败",
					"Version":         "",
					"Clients":         0,
					"TrafficIn":       "",
					"TrafficOut":      "",
					"TotalTrafficIn":  "",
					"TotalTrafficOut": "",
					"Load":            "0.00%",
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
					"NodeName":        node.NodeName,
					"Status":          "offline",
					"Error":           "响应解析失败",
					"Version":         "",
					"Clients":         0,
					"TrafficIn":       "",
					"TrafficOut":      "",
					"TotalTrafficIn":  "",
					"TotalTrafficOut": "",
					"Load":            "0.00%",
				}
				mu.Unlock()
				return
			}

			// 格式化流量数据
			trafficIn := formatTraffic(nodeInfo.TotalTrafficIn)
			trafficOut := formatTraffic(nodeInfo.TotalTrafficOut)

			// 获取总流量数据
			totalTrafficIn := ""
			totalTrafficOut := ""
			if trafficLog, ok := nodeTrafficMap[node.NodeName]; ok && trafficLog != nil {
				totalTrafficIn = formatTraffic(trafficLog.TrafficIn)
				totalTrafficOut = formatTraffic(trafficLog.TrafficOut)
			}

			// 获取节点负载信息
			loadKey := fmt.Sprintf("node:load:%s", node.IP)
			loadValue, err := h.redisClient.Get(ctx, loadKey).Result()
			if err != nil && err != redis.Nil {
				h.logger.Error("Failed to get node load from Redis", "error", err, "ip", node.IP)
			}

			// 如果没有找到负载信息，设置为默认值
			if err == redis.Nil {
				loadValue = "N/A"
			}

			// 保存结果
			mu.Lock()
			results[strconv.FormatInt(node.ID, 10)] = gin.H{
				"NodeName":        node.NodeName,
				"Status":          "online",
				"Version":         nodeInfo.Version,
				"Clients":         nodeInfo.ClientCounts,
				"TrafficIn":       trafficIn,
				"TrafficOut":      trafficOut,
				"TotalTrafficIn":  totalTrafficIn,
				"TotalTrafficOut": totalTrafficOut,
				"Load":            loadValue,
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
		"summary": gin.H{
			"AllNodesTrafficIn":  totalInFormatted,
			"AllNodesTrafficOut": totalOutFormatted,
		},
	})
}

// DonateNodeRequest 捐赠节点请求参数
type DonateNodeRequest struct {
	NodeName     string   `json:"node_name" binding:"required"`
	Description  string   `json:"description" binding:"required"`
	Bandwidth    string   `json:"bandwidth" binding:"required"` // 节点带宽
	IP           string   `json:"ip" binding:"required"`
	FrpsPort     int      `json:"server_port" binding:"required"` // 服务端口
	URL          string   `json:"panel_url" binding:"required"`   // 面板地址
	PortRange    string   `json:"port_range" binding:"required"`
	AllowedTypes []string `json:"allowed_types" binding:"required"` // 允许类型
	Permission   []string `json:"permission" binding:"required"`    // 允许访问的权限组
	Token        string   `json:"token" binding:"required"`         // frps的Token
	User         string   `json:"user" binding:"required"`          // frps的用户名
}

// DonateNode 捐赠节点
func (h *NodeHandler) DonateNode(c *gin.Context) {
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

	// 解析请求
	var req DonateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("捐赠节点参数绑定失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	// 将节点名称与带宽结合
	combinedNodeName := fmt.Sprintf("%s-%s", req.NodeName, req.Bandwidth)

	// 检查节点名是否已存在
	existingNode, err := h.nodeService.GetByNodeName(context.Background(), combinedNodeName)
	if err == nil && existingNode != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "节点名已存在"})
		return
	}

	// 将AllowedTypes转换为JSON字符串
	allowedTypesBytes, err := json.Marshal(req.AllowedTypes)
	if err != nil {
		h.logger.Error("转换AllowedTypes失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}
	allowedTypesStr := string(allowedTypesBytes)

	// 将Permission转换为JSON字符串
	permissionBytes, err := json.Marshal(req.Permission)
	if err != nil {
		h.logger.Error("转换Permission失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}
	permissionStr := string(permissionBytes)

	// 将节点描述转换为JSON数组格式
	// 如果用户输入的是简单字符串，将其转换为单元素数组
	descriptionArray := []string{req.Description}
	descriptionBytes, err := json.Marshal(descriptionArray)
	if err != nil {
		h.logger.Error("转换Description失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}
	descriptionStr := string(descriptionBytes)

	// 创建节点
	node := &repository.Node{
		NodeName:     combinedNodeName, // 使用组合后的节点名称
		FrpsPort:     req.FrpsPort,
		URL:          req.URL,
		Token:        req.Token,                                           // 使用用户提供的Token
		User:         req.User,                                            // 使用用户提供的User
		Description:  sql.NullString{String: descriptionStr, Valid: true}, // 使用JSON数组格式的描述
		Permission:   permissionStr,
		AllowedTypes: allowedTypesStr,
		Host:         sql.NullString{String: "", Valid: false}, // host字段保持为空
		PortRange:    req.PortRange,
		IP:           req.IP,
		Status:       2, // 设置为待审核状态(2)
	}

	// 如果是用户捐赠，设置所有者ID
	if user != nil && user.ID > 0 {
		node.OwnerID = sql.NullInt64{Int64: user.ID, Valid: true}
	}

	// 保存节点
	err = h.nodeService.CreateNode(context.Background(), node)
	if err != nil {
		h.logger.Error("创建捐赠节点失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "创建捐赠节点失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "节点捐赠成功，请等待管理员审核",
		"data": gin.H{
			"node_id":   node.ID,
			"node_name": node.NodeName,
			"status":    "待审核",
		},
	})
}

// GetUserNodes 获取用户自己的节点列表
func (h *NodeHandler) GetUserNodes(c *gin.Context) {
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

	// 获取分页参数
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

	// 获取用户的所有节点列表
	allNodes, err := h.nodeService.GetNodesByOwnerID(context.Background(), user.ID)
	if err != nil {
		h.logger.Error("获取用户节点失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	// 计算总数和总页数
	total := len(allNodes)
	pages := (total + pageSize - 1) / pageSize

	// 计算当前页的起始索引和结束索引
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}

	// 分页获取当前页的节点
	var pageNodes []*repository.Node
	if start < total {
		pageNodes = allNodes[start:end]
	} else {
		pageNodes = []*repository.Node{}
	}

	// 构建返回数据
	var nodeList []gin.H
	for _, node := range pageNodes {
		// 解析AllowedTypes字段，它是JSON格式的字符串
		var allowedTypes []string
		if err := json.Unmarshal([]byte(node.AllowedTypes), &allowedTypes); err != nil {
			// 如果解析失败，使用原始字符串
			h.logger.Error("解析allowed_types失败", "error", err, "value", node.AllowedTypes)
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

		// 获取节点状态描述
		var statusDesc string
		switch node.Status {
		case 0:
			statusDesc = "禁用"
		case 1:
			statusDesc = "启用"
		case 2:
			statusDesc = "待审核"
		default:
			statusDesc = "未知"
		}

		nodeList = append(nodeList, gin.H{
			"id":            node.ID,
			"node_name":     node.NodeName,
			"allowed_types": allowedTypes,
			"port_range":    node.PortRange,
			"description":   description,
			"status":        node.Status,
			"status_desc":   statusDesc,
			"created_at":    node.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取成功",
		"pagination": gin.H{
			"page":      page,
			"page_size": pageSize,
			"pages":     pages,
			"total":     total,
		},
		"nodes": nodeList,
	})
}

// NodeLoadInfo 节点负载信息
type NodeLoadInfo struct {
	IP                string  `json:"ip"`
	LoadScore         float64 `json:"load_score"`
	CurrentConns      int     `json:"current_conns"`
	PeakConns         int     `json:"peak_conns"`
	CurrentTraffic    int64   `json:"current_traffic"`
	PeakTraffic       int64   `json:"peak_traffic"`
	CPUUsage          float64 `json:"cpu_usage"`
	MemUsage          float64 `json:"mem_usage"`
	ConnGrowthRate    float64 `json:"conn_growth_rate"`
	TrafficGrowthRate float64 `json:"traffic_growth_rate"`
	Timestamp         int64   `json:"timestamp"`
}

// ReceiveNodeLoad 接收节点负载信息
func (h *NodeHandler) ReceiveNodeLoad(c *gin.Context) {
	var nodeLoad NodeLoadInfo
	if err := c.ShouldBindJSON(&nodeLoad); err != nil {
		h.logger.Error("Failed to bind node load info", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无效的请求数据"})
		return
	}

	// 将负载百分比转换为字符串形式，如0.11118003913894325 -> "0.11%"
	loadPercentage := fmt.Sprintf("%.2f%%", nodeLoad.LoadScore*100)

	// 存储到Redis中，使用IP作为键
	ctx := context.Background()
	key := fmt.Sprintf("node:load:%s", nodeLoad.IP)

	// 设置过期时间为60秒，确保数据不会过期
	err := h.redisClient.Set(ctx, key, loadPercentage, 60*time.Second).Err()
	if err != nil {
		h.logger.Error("Failed to store node load in Redis", "error", err, "ip", nodeLoad.IP)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "存储负载信息失败"})
		return
	}

	h.logger.Info("Node load info received", "ip", nodeLoad.IP, "load", loadPercentage)
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "负载信息已接收"})
}
