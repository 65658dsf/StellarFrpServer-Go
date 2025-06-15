package admin

import (
	"bytes"
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
	"github.com/redis/go-redis/v9"
)

// 辅助函数，返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ProxyAdminHandler 隧道管理处理器
type ProxyAdminHandler struct {
	proxyService service.ProxyService
	nodeService  service.NodeService
	userService  service.UserService
	redisCli     *redis.Client
	logger       *logger.Logger
}

// NewProxyAdminHandler 创建隧道管理处理器实例
func NewProxyAdminHandler(
	proxyService service.ProxyService,
	nodeService service.NodeService,
	userService service.UserService,
	redisCli *redis.Client,
	logger *logger.Logger,
) *ProxyAdminHandler {
	return &ProxyAdminHandler{
		proxyService: proxyService,
		nodeService:  nodeService,
		userService:  userService,
		redisCli:     redisCli,
		logger:       logger,
	}
}

// ListProxies 获取所有隧道列表（带分页）
func (h *ProxyAdminHandler) ListProxies(c *gin.Context) {
	// 处理分页参数
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	statusFilter := c.Query("status") // 可选的状态过滤

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

	var proxies []*repository.Proxy
	var total int

	// 根据状态过滤条件选择查询方法
	if statusFilter != "" && statusFilter != "all" {
		// 使用状态过滤直接从数据库查询
		proxies, err = h.proxyService.ListByStatus(context.Background(), statusFilter, offset, pageSize)
		if err != nil {
			h.logger.Error("获取隧道列表失败", "error", err, "status", statusFilter)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取隧道列表失败"})
			return
		}

		// 获取此状态的隧道总数
		total, err = h.proxyService.CountByStatus(context.Background(), statusFilter)
		if err != nil {
			h.logger.Error("获取隧道总数失败", "error", err, "status", statusFilter)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取隧道总数失败"})
			return
		}
	} else {
		// 获取所有隧道列表
		proxies, err = h.proxyService.List(context.Background(), offset, pageSize)
		if err != nil {
			h.logger.Error("获取隧道列表失败", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取隧道列表失败"})
			return
		}

		// 获取隧道总数
		total, err = h.proxyService.Count(context.Background())
		if err != nil {
			h.logger.Error("获取隧道总数失败", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取隧道总数失败"})
			return
		}
	}

	// 计算总页数
	pages := (total + pageSize - 1) / pageSize

	// 获取所有节点信息（用于展示节点名称）
	allNodes, err := h.nodeService.GetAllNodes(context.Background())
	if err != nil {
		h.logger.Error("获取节点列表失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点列表失败"})
		return
	}

	// 构建节点ID到节点名称的映射
	nodeMap := make(map[int64]*repository.Node)
	for _, node := range allNodes {
		nodeMap[node.ID] = node
	}

	// 构建返回结果
	var result []gin.H
	for _, proxy := range proxies {
		node, exists := nodeMap[proxy.Node]
		nodeName := "未知节点"
		if exists {
			nodeName = node.NodeName
		}

		result = append(result, gin.H{
			"id":                  proxy.ID,
			"username":            proxy.Username,
			"proxy_name":          proxy.ProxyName,
			"proxy_type":          proxy.ProxyType,
			"local_ip":            proxy.LocalIP,
			"local_port":          proxy.LocalPort,
			"use_encryption":      proxy.UseEncryption,
			"use_compression":     proxy.UseCompression,
			"domain":              proxy.Domain,
			"host_header_rewrite": proxy.HostHeaderRewrite,
			"remote_port":         proxy.RemotePort,
			"header_x_from_where": proxy.HeaderXFromWhere,
			"status":              proxy.Status,
			"lastupdate":          proxy.LastUpdate,
			"node_id":             proxy.Node,
			"node_name":           nodeName,
			"run_id":              proxy.RunID,
			"traffic_quota":       proxy.TrafficQuota,
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
		"proxies": result,
	})
}

// GetProxyStatus 获取隧道状态
func (h *ProxyAdminHandler) GetProxyStatus(c *gin.Context) {
	// 获取隧道ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无效的隧道ID"})
		return
	}

	// 获取隧道信息
	proxy, err := h.proxyService.GetByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "隧道不存在"})
		return
	}

	// 获取节点信息
	node, err := h.nodeService.GetByID(context.Background(), proxy.Node)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "节点不存在"})
		return
	}

	// 检查缓存中是否有状态信息
	cacheKey := fmt.Sprintf("proxy:status:%d", id)
	cachedStatus, err := h.redisCli.Get(context.Background(), cacheKey).Result()
	if err == nil {
		// 缓存命中，解析并返回
		var statusData gin.H
		if err := json.Unmarshal([]byte(cachedStatus), &statusData); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"code": 200,
				"msg":  "获取成功",
				"data": statusData,
			})
			return
		}
	}

	// 缓存未命中，向节点查询状态
	// 根据隧道类型构建API URL
	apiURL := fmt.Sprintf("%s/api/proxy/%s", node.URL, proxy.ProxyType)
	h.logger.Debug("请求节点API", "url", apiURL, "node_id", node.ID, "proxy_type", proxy.ProxyType)
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		h.logger.Error("创建请求失败", "error", err, "url", apiURL)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "获取隧道状态失败",
			"data": gin.H{
				"status":          "offline",
				"proxy_id":        proxy.ID,
				"proxy_name":      proxy.ProxyName,
				"node_id":         node.ID,
				"node_name":       node.NodeName,
				"last_start_time": "",
				"last_close_time": "",
				"cur_conns":       0,
				"traffic_in":      0,
				"traffic_out":     0,
			},
		})
		return
	}

	req.SetBasicAuth(node.User, node.Token)
	resp, err := client.Do(req)
	if err != nil {
		h.logger.Error("请求节点失败", "error", err, "nodeID", node.ID)
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"msg":  "获取成功，但节点离线",
			"data": gin.H{
				"status":          "offline",
				"proxy_id":        proxy.ID,
				"proxy_name":      proxy.ProxyName,
				"node_id":         node.ID,
				"node_name":       node.NodeName,
				"last_start_time": "",
				"last_close_time": "",
				"cur_conns":       0,
				"traffic_in":      0,
				"traffic_out":     0,
			},
		})
		return
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("读取响应内容失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "读取节点响应失败",
		})
		return
	}

	// 记录原始响应数据进行调试
	h.logger.Debug("节点原始响应", "nodeID", node.ID, "body_length", len(body))

	// 解析响应
	var proxyTypeResponse struct {
		Proxies []map[string]interface{} `json:"proxies"`
	}
	if err := json.Unmarshal(body, &proxyTypeResponse); err != nil {
		h.logger.Error("解析节点响应失败", "error", err, "body_preview", string(body[:min(100, len(body))]))
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"msg":  "获取成功，但无法解析节点响应",
			"data": gin.H{
				"status":          "offline",
				"proxy_id":        proxy.ID,
				"proxy_name":      proxy.ProxyName,
				"node_id":         node.ID,
				"node_name":       node.NodeName,
				"last_start_time": "",
				"last_close_time": "",
				"cur_conns":       0,
				"traffic_in":      0,
				"traffic_out":     0,
			},
		})
		return
	}

	// 预期的隧道名称格式
	expectedProxyName := proxy.Username + "." + proxy.ProxyName

	// 获取隧道状态信息
	status := "offline"
	lastStartTime := ""
	lastCloseTime := ""
	curConns := 0
	trafficIn := int64(0)
	trafficOut := int64(0)

	// 在返回的隧道列表中查找目标隧道
	for _, proxyInfo := range proxyTypeResponse.Proxies {
		name, ok := proxyInfo["name"].(string)
		if !ok || name != expectedProxyName {
			continue
		}

		// 找到目标隧道
		if statusStr, ok := proxyInfo["status"].(string); ok && statusStr == "online" {
			status = "online"
		}

		if start, ok := proxyInfo["lastStartTime"].(string); ok {
			lastStartTime = start
		}

		if close, ok := proxyInfo["lastCloseTime"].(string); ok {
			lastCloseTime = close
		}

		if conns, ok := proxyInfo["curConns"].(float64); ok {
			curConns = int(conns)
		}

		if in, ok := proxyInfo["todayTrafficIn"].(float64); ok {
			trafficIn = int64(in)
		}

		if out, ok := proxyInfo["todayTrafficOut"].(float64); ok {
			trafficOut = int64(out)
		}

		break
	}

	// 构建响应数据
	statusData := gin.H{
		"status":          status,
		"proxy_id":        proxy.ID,
		"proxy_name":      proxy.ProxyName,
		"node_id":         node.ID,
		"node_name":       node.NodeName,
		"last_start_time": lastStartTime,
		"last_close_time": lastCloseTime,
		"cur_conns":       curConns,
		"traffic_in":      trafficIn,
		"traffic_out":     trafficOut,
	}

	// 缓存状态数据（30秒）
	statusJSON, _ := json.Marshal(statusData)
	h.redisCli.Set(context.Background(), cacheKey, statusJSON, 30*time.Second)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取成功",
		"data": statusData,
	})
}

// ListAllProxiesStatus 批量获取所有隧道状态（带分页，优化查询）
func (h *ProxyAdminHandler) ListAllProxiesStatus(c *gin.Context) {
	// 可选的查询参数：仅查询在线隧道
	onlineOnly := c.DefaultQuery("online_only", "false") == "true"

	// 处理分页参数
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

	// 获取所有隧道（如果onlineOnly为true，则仅获取在线隧道）
	var proxies []*repository.Proxy
	var total int

	if onlineOnly {
		proxies, err = h.proxyService.ListByStatus(context.Background(), "online", offset, pageSize)
		if err != nil {
			h.logger.Error("获取隧道列表失败", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取隧道列表失败"})
			return
		}

		// 获取在线隧道总数
		total, err = h.proxyService.CountByStatus(context.Background(), "online")
	} else {
		proxies, err = h.proxyService.List(context.Background(), offset, pageSize)
		if err != nil {
			h.logger.Error("获取隧道列表失败", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取隧道列表失败"})
			return
		}

		// 获取隧道总数
		total, err = h.proxyService.Count(context.Background())
	}

	if err != nil {
		h.logger.Error("获取隧道总数失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取隧道总数失败"})
		return
	}

	// 计算总页数
	pages := (total + pageSize - 1) / pageSize

	if len(proxies) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"msg":  "获取成功，但没有隧道",
			"data": gin.H{},
			"pagination": gin.H{
				"page":      page,
				"page_size": pageSize,
				"pages":     pages,
				"total":     total,
			},
		})
		return
	}

	// 按节点ID分组隧道
	nodeProxiesMap := make(map[int64][]*repository.Proxy)
	for _, proxy := range proxies {
		nodeProxiesMap[proxy.Node] = append(nodeProxiesMap[proxy.Node], proxy)
	}

	// 获取所有涉及的节点信息
	nodeIDs := make([]int64, 0, len(nodeProxiesMap))
	for nodeID := range nodeProxiesMap {
		nodeIDs = append(nodeIDs, nodeID)
	}

	// 获取节点信息
	allNodes, err := h.nodeService.GetAllNodes(context.Background())
	if err != nil {
		h.logger.Error("获取节点列表失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点列表失败"})
		return
	}

	// 创建节点ID到节点的映射
	nodeMap := make(map[int64]*repository.Node)
	for _, node := range allNodes {
		nodeMap[node.ID] = node
	}

	// 同步获取所有节点上的隧道状态
	var mu sync.Mutex
	proxyStatusMap := make(map[int64]gin.H)
	var wg sync.WaitGroup

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for nodeID, nodeProxies := range nodeProxiesMap {
		node, exists := nodeMap[nodeID]
		if !exists {
			// 节点不存在，标记所有相关隧道为离线
			for _, proxy := range nodeProxies {
				proxyStatusMap[proxy.ID] = gin.H{
					"status":          "offline",
					"proxy_id":        proxy.ID,
					"proxy_name":      proxy.ProxyName,
					"node_id":         nodeID,
					"node_name":       "未知节点",
					"last_start_time": "",
					"last_close_time": "",
					"cur_conns":       0,
					"traffic_in":      0,
					"traffic_out":     0,
				}
			}
			continue
		}

		wg.Add(1)
		go func(node *repository.Node, proxies []*repository.Proxy) {
			defer wg.Done()

			// 构建批量查询URL
			// 需要按照隧道类型分别查询
			proxyTypeMap := make(map[string][]*repository.Proxy)
			for _, proxy := range proxies {
				proxyType := proxy.ProxyType
				proxyTypeMap[proxyType] = append(proxyTypeMap[proxyType], proxy)
			}

			// 存储所有隧道的状态信息
			proxyStatusInfoMap := make(map[string]map[string]interface{})

			// 对每种隧道类型分别查询
			for proxyType, _ := range proxyTypeMap {
				apiURL := fmt.Sprintf("%s/api/proxy/%s", node.URL, proxyType)
				h.logger.Debug("请求节点API", "url", apiURL, "node_id", node.ID, "proxy_type", proxyType)

				req, err := http.NewRequest("GET", apiURL, nil)
				if err != nil {
					h.logger.Error("创建请求失败", "error", err, "url", apiURL)
					continue
				}

				req.SetBasicAuth(node.User, node.Token)
				resp, err := client.Do(req)
				if err != nil {
					h.logger.Error("请求节点失败", "error", err, "nodeID", node.ID, "proxy_type", proxyType)
					continue
				}

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()

				if err != nil {
					h.logger.Error("读取响应内容失败", "error", err, "nodeID", node.ID, "proxy_type", proxyType)
					continue
				}

				// 记录原始响应数据进行调试
				h.logger.Debug("节点原始响应", "nodeID", node.ID, "proxy_type", proxyType, "body_length", len(body))

				// 解析节点响应
				var proxyTypeResponse struct {
					Proxies []map[string]interface{} `json:"proxies"`
				}

				if err := json.Unmarshal(body, &proxyTypeResponse); err != nil {
					h.logger.Error("解析节点响应失败", "error", err, "nodeID", node.ID, "proxy_type", proxyType, "body_preview", string(body[:min(100, len(body))]))
					continue
				}

				// 处理该类型的所有隧道
				for _, proxyInfo := range proxyTypeResponse.Proxies {
					name, ok := proxyInfo["name"].(string)
					if !ok {
						continue
					}
					proxyStatusInfoMap[name] = proxyInfo
				}
			}

			// 处理节点返回的隧道状态
			mu.Lock()
			for _, proxy := range proxies {
				// 预期的隧道名称格式
				expectedProxyName := proxy.Username + "." + proxy.ProxyName

				// 默认离线状态
				status := "offline"
				lastStartTime := ""
				lastCloseTime := ""
				curConns := 0
				trafficIn := int64(0)
				trafficOut := int64(0)

				// 查找并解析隧道状态
				if proxyInfo, found := proxyStatusInfoMap[expectedProxyName]; found {
					if statusStr, ok := proxyInfo["status"].(string); ok && statusStr == "online" {
						status = "online"
					}

					if start, ok := proxyInfo["lastStartTime"].(string); ok {
						lastStartTime = start
					}

					if close, ok := proxyInfo["lastCloseTime"].(string); ok {
						lastCloseTime = close
					}

					if conns, ok := proxyInfo["curConns"].(float64); ok {
						curConns = int(conns)
					}

					if in, ok := proxyInfo["todayTrafficIn"].(float64); ok {
						trafficIn = int64(in)
					}

					if out, ok := proxyInfo["todayTrafficOut"].(float64); ok {
						trafficOut = int64(out)
					}
				}

				// 保存状态结果
				proxyStatusMap[proxy.ID] = gin.H{
					"status":          status,
					"proxy_id":        proxy.ID,
					"proxy_name":      proxy.ProxyName,
					"node_id":         node.ID,
					"node_name":       node.NodeName,
					"last_start_time": lastStartTime,
					"last_close_time": lastCloseTime,
					"cur_conns":       curConns,
					"traffic_in":      trafficIn,
					"traffic_out":     trafficOut,
				}

				// 缓存该隧道状态（30秒）
				cacheKey := fmt.Sprintf("proxy:status:%d", proxy.ID)
				statusJSON, _ := json.Marshal(proxyStatusMap[proxy.ID])
				h.redisCli.Set(context.Background(), cacheKey, statusJSON, 30*time.Second)
			}
			mu.Unlock()
		}(node, nodeProxies)
	}

	// 等待所有查询完成
	wg.Wait()

	// 按原隧道顺序构建结果
	var result []gin.H
	for _, proxy := range proxies {
		if status, exists := proxyStatusMap[proxy.ID]; exists {
			// 如果要筛选仅显示在线隧道
			if onlineOnly {
				if statusStr, ok := status["status"].(string); ok && statusStr != "online" {
					continue
				}
			}
			result = append(result, status)
		}
	}

	// 如果进行了状态过滤，更新总数
	filteredTotal := len(result)
	if onlineOnly {
		pages = (filteredTotal + pageSize - 1) / pageSize
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取成功",
		"pagination": gin.H{
			"page":      page,
			"page_size": pageSize,
			"pages":     pages,
			"total":     filteredTotal,
		},
		"proxies": result,
	})
}

// SearchProxies 搜索隧道
func (h *ProxyAdminHandler) SearchProxies(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "搜索关键词不能为空"})
		return
	}

	// 从数据库获取所有隧道
	allProxies, err := h.proxyService.List(context.Background(), 0, 1000)
	if err != nil {
		h.logger.Error("获取隧道列表失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "搜索隧道失败"})
		return
	}

	// 获取所有节点信息
	allNodes, err := h.nodeService.GetAllNodes(context.Background())
	if err != nil {
		h.logger.Error("获取节点列表失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "搜索隧道失败"})
		return
	}

	// 构建节点映射
	nodeMap := make(map[int64]*repository.Node)
	for _, node := range allNodes {
		nodeMap[node.ID] = node
	}

	// 搜索匹配关键词的隧道
	var matchedProxies []*repository.Proxy
	for _, proxy := range allProxies {
		// 检查隧道名称、用户名、隧道类型是否包含关键词
		if containsIgnoreCase(proxy.ProxyName, keyword) ||
			containsIgnoreCase(proxy.Username, keyword) ||
			containsIgnoreCase(proxy.ProxyType, keyword) {
			matchedProxies = append(matchedProxies, proxy)
		}
	}

	// 构建返回结果
	var result []gin.H
	for _, proxy := range matchedProxies {
		node, exists := nodeMap[proxy.Node]
		nodeName := "未知节点"
		if exists {
			nodeName = node.NodeName
		}

		result = append(result, gin.H{
			"id":                  proxy.ID,
			"username":            proxy.Username,
			"proxy_name":          proxy.ProxyName,
			"proxy_type":          proxy.ProxyType,
			"local_ip":            proxy.LocalIP,
			"local_port":          proxy.LocalPort,
			"use_encryption":      proxy.UseEncryption,
			"use_compression":     proxy.UseCompression,
			"domain":              proxy.Domain,
			"host_header_rewrite": proxy.HostHeaderRewrite,
			"remote_port":         proxy.RemotePort,
			"header_x_from_where": proxy.HeaderXFromWhere,
			"status":              proxy.Status,
			"lastupdate":          proxy.LastUpdate,
			"node_id":             proxy.Node,
			"node_name":           nodeName,
			"run_id":              proxy.RunID,
			"traffic_quota":       proxy.TrafficQuota,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"msg":     "搜索成功",
		"total":   len(result),
		"proxies": result,
	})
}

// CloseProxy 管理员关闭用户隧道
func (h *ProxyAdminHandler) CloseProxy(c *gin.Context) {
	// 获取隧道ID
	type CloseRequest struct {
		ID int64 `json:"id" binding:"required"`
	}

	var req CloseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误：" + err.Error()})
		return
	}

	// 获取隧道信息
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

	// 检查隧道是否已经离线
	if proxy.RunID == "" || proxy.Status == "offline" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "该隧道已经是离线状态"})
		return
	}

	// 获取节点信息
	node, err := h.nodeService.GetByID(context.Background(), proxy.Node)
	if err != nil {
		h.logger.Error("获取节点信息失败", "error", err, "nodeID", proxy.Node)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点信息失败"})
		return
	}

	// 构建关闭隧道的请求
	requestBody, err := json.Marshal(map[string]string{
		"runid": proxy.RunID,
	})
	if err != nil {
		h.logger.Error("构建请求体失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	// 发送关闭请求到节点
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
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "节点返回错误: " + string(body)})
		return
	}

	// 更新隧道状态
	proxy.RunID = ""
	proxy.Status = "offline"
	proxy.LastUpdate = time.Now().Format("2006-01-02 15:04:05")
	if err := h.proxyService.Update(context.Background(), proxy); err != nil {
		h.logger.Error("更新隧道状态失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "隧道已关闭，但更新数据库状态失败"})
		return
	}

	// 清除状态缓存
	cacheKey := fmt.Sprintf("proxy:status:%d", proxy.ID)
	h.redisCli.Del(context.Background(), cacheKey)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "隧道已成功关闭",
		"data": gin.H{
			"id":            proxy.ID,
			"proxy_name":    proxy.ProxyName,
			"username":      proxy.Username,
			"node_id":       proxy.Node,
			"node_name":     node.NodeName,
			"last_update":   proxy.LastUpdate,
			"current_state": "offline",
		},
	})
}

// CloseUserProxies 管理员关闭指定用户的所有隧道
func (h *ProxyAdminHandler) CloseUserProxies(c *gin.Context) {
	// 获取用户名
	type CloseUserRequest struct {
		Username string `json:"username" binding:"required"`
	}

	var req CloseUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误：" + err.Error()})
		return
	}

	// 获取用户信息，验证用户是否存在
	user, err := h.userService.GetByUsername(context.Background(), req.Username)
	if err != nil {
		h.logger.Error("获取用户信息失败", "error", err, "username", req.Username)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取用户信息失败"})
		return
	}

	if user == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "用户不存在"})
		return
	}

	// 获取用户的所有隧道
	proxies, err := h.proxyService.GetByUsername(context.Background(), req.Username)
	if err != nil {
		h.logger.Error("获取用户隧道列表失败", "error", err, "username", req.Username)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取用户隧道列表失败"})
		return
	}

	if len(proxies) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "该用户没有隧道"})
		return
	}

	// 获取所有节点信息，用于后续批量处理
	allNodes, err := h.nodeService.GetAllNodes(context.Background())
	if err != nil {
		h.logger.Error("获取节点列表失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点列表失败"})
		return
	}

	// 构建节点ID到节点的映射
	nodeMap := make(map[int64]*repository.Node)
	for _, node := range allNodes {
		nodeMap[node.ID] = node
	}

	// 并发关闭用户的所有隧道
	var wg sync.WaitGroup
	var mu sync.Mutex
	result := make(map[int64]gin.H)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for _, proxy := range proxies {
		// 只处理在线的隧道
		if proxy.Status != "online" || proxy.RunID == "" {
			mu.Lock()
			result[proxy.ID] = gin.H{
				"id":            proxy.ID,
				"proxy_name":    proxy.ProxyName,
				"node_id":       proxy.Node,
				"status":        "已经是离线状态",
				"close_success": false,
			}
			mu.Unlock()
			continue
		}

		node, exists := nodeMap[proxy.Node]
		if !exists {
			mu.Lock()
			result[proxy.ID] = gin.H{
				"id":            proxy.ID,
				"proxy_name":    proxy.ProxyName,
				"node_id":       proxy.Node,
				"status":        "节点不存在",
				"close_success": false,
			}
			mu.Unlock()
			continue
		}

		wg.Add(1)
		go func(proxy *repository.Proxy, node *repository.Node) {
			defer wg.Done()

			// 构建关闭请求
			requestBody, err := json.Marshal(map[string]string{
				"runid": proxy.RunID,
			})
			if err != nil {
				h.logger.Error("构建请求体失败", "error", err, "proxy_id", proxy.ID)
				mu.Lock()
				result[proxy.ID] = gin.H{
					"id":            proxy.ID,
					"proxy_name":    proxy.ProxyName,
					"node_id":       proxy.Node,
					"node_name":     node.NodeName,
					"status":        "构建请求失败",
					"close_success": false,
					"error":         err.Error(),
				}
				mu.Unlock()
				return
			}

			// 发送关闭请求
			apiURL := fmt.Sprintf("%s/api/client/kick", node.URL)
			req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
			if err != nil {
				h.logger.Error("创建请求失败", "error", err, "url", apiURL)
				mu.Lock()
				result[proxy.ID] = gin.H{
					"id":            proxy.ID,
					"proxy_name":    proxy.ProxyName,
					"node_id":       proxy.Node,
					"node_name":     node.NodeName,
					"status":        "创建请求失败",
					"close_success": false,
					"error":         err.Error(),
				}
				mu.Unlock()
				return
			}

			req.Header.Set("Content-Type", "application/json")
			req.SetBasicAuth(node.User, node.Token)

			resp, err := client.Do(req)
			if err != nil {
				h.logger.Error("发送请求失败", "error", err, "url", apiURL)
				mu.Lock()
				result[proxy.ID] = gin.H{
					"id":            proxy.ID,
					"proxy_name":    proxy.ProxyName,
					"node_id":       proxy.Node,
					"node_name":     node.NodeName,
					"status":        "发送请求失败",
					"close_success": false,
					"error":         err.Error(),
				}
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			// 处理响应
			success := resp.StatusCode == http.StatusOK

			// 更新隧道状态
			if success {
				proxy.RunID = ""
				proxy.Status = "offline"
				proxy.LastUpdate = time.Now().Format("2006-01-02 15:04:05")
				err = h.proxyService.Update(context.Background(), proxy)

				// 清除状态缓存
				cacheKey := fmt.Sprintf("proxy:status:%d", proxy.ID)
				h.redisCli.Del(context.Background(), cacheKey)
			}

			mu.Lock()
			statusText := "关闭失败"
			errText := ""
			if success {
				statusText = "已关闭"
			}
			if err != nil {
				errText = err.Error()
			}
			result[proxy.ID] = gin.H{
				"id":            proxy.ID,
				"proxy_name":    proxy.ProxyName,
				"node_id":       proxy.Node,
				"node_name":     node.NodeName,
				"status":        statusText,
				"close_success": success,
				"error":         errText,
			}
			mu.Unlock()
		}(proxy, node)
	}

	// 等待所有关闭操作完成
	wg.Wait()

	// 统计关闭结果
	successCount := 0
	failCount := 0
	for _, r := range result {
		if r["close_success"].(bool) {
			successCount++
		} else {
			failCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  fmt.Sprintf("操作完成：成功关闭 %d 个隧道，失败 %d 个", successCount, failCount),
		"data": gin.H{
			"username":      req.Username,
			"total_proxies": len(proxies),
			"success_count": successCount,
			"fail_count":    failCount,
			"details":       result,
		},
	})
}

// DeleteProxy 管理员删除隧道
func (h *ProxyAdminHandler) DeleteProxy(c *gin.Context) {
	// 获取隧道ID
	type DeleteRequest struct {
		ID int64 `json:"id" binding:"required"`
	}

	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误：" + err.Error()})
		return
	}

	// 获取隧道信息
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

	// 如果隧道处于在线状态，先关闭隧道
	if proxy.Status == "online" && proxy.RunID != "" {
		// 获取节点信息
		node, err := h.nodeService.GetByID(context.Background(), proxy.Node)
		if err != nil {
			h.logger.Error("获取节点信息失败", "error", err, "nodeID", proxy.Node)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取节点信息失败"})
			return
		}

		// 构建关闭隧道的请求
		requestBody, err := json.Marshal(map[string]string{
			"runid": proxy.RunID,
		})
		if err != nil {
			h.logger.Error("构建请求体失败", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
			return
		}

		// 发送关闭请求到节点
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
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "关闭隧道请求失败，无法删除"})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			h.logger.Error("读取响应失败", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "读取响应失败"})
			return
		}

		// 尝试解析响应，但即使解析失败也继续执行删除操作
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			h.logger.Warn("解析节点响应失败，但将继续删除隧道", "error", err, "body", string(body))
		} else if resp.StatusCode != http.StatusOK {
			h.logger.Warn("节点返回非200状态码，但将继续删除隧道", "statusCode", resp.StatusCode, "response", string(body))
		}
	}

	// 删除隧道
	err = h.proxyService.Delete(context.Background(), req.ID)
	if err != nil {
		h.logger.Error("删除隧道失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "删除隧道失败: " + err.Error()})
		return
	}

	// 清除状态缓存
	cacheKey := fmt.Sprintf("proxy:status:%d", proxy.ID)
	h.redisCli.Del(context.Background(), cacheKey)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "隧道已成功删除",
		"data": gin.H{
			"id":         proxy.ID,
			"proxy_name": proxy.ProxyName,
			"username":   proxy.Username,
		},
	})
}
