package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ProductHandler 商品处理器
type ProductHandler struct {
	productService *service.ProductService
	logger         *logger.Logger
}

// NewProductHandler 创建商品处理器
func NewProductHandler(productService *service.ProductService, logger *logger.Logger) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		logger:         logger,
	}
}

// GetProducts 获取所有商品
func (h *ProductHandler) GetProducts(c *gin.Context) {
	products, err := h.productService.GetProducts()
	if err != nil {
		h.logger.Error("获取商品列表失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "获取商品列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取商品列表成功",
		"data":    products,
	})
}

// CreateOrderLink 创建订单链接
func (h *ProductHandler) CreateOrderLink(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"code":    401,
			"message": "用户未认证",
		})
		return
	}

	// 解析JSON请求体
	var requestBody struct {
		ProductID uint64 `json:"product_id" binding:"required"`
		Remark    string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    400,
			"message": "无效的请求参数",
		})
		return
	}

	// 将userID转换为uint64
	var userIDUint uint64
	switch v := userID.(type) {
	case int64:
		userIDUint = uint64(v)
	case uint64:
		userIDUint = v
	case float64:
		userIDUint = uint64(v)
	case int:
		userIDUint = uint64(v)
	default:
		h.logger.Error("无效的用户ID类型", "type", fmt.Sprintf("%T", userID), "value", userID)
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "服务器内部错误",
		})
		return
	}

	orderLink, orderNo, err := h.productService.CreateOrderLink(userIDUint, requestBody.ProductID, requestBody.Remark)
	if err != nil {
		h.logger.Error("创建订单链接失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "创建订单链接失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "创建订单链接成功",
		"data": gin.H{
			"order_link": orderLink,
			"order_no":   orderNo,
		},
	})
}

// AfdianWebhook 处理爱发电Webhook回调
func (h *ProductHandler) AfdianWebhook(c *gin.Context) {
	// 读取请求体
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("读取请求体失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"ec": 400,
			"em": "读取请求体失败",
		})
		return
	}

	// 记录请求内容
	h.logger.Info("收到爱发电Webhook请求", "body", string(body))

	// 解析JSON数据
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		h.logger.Error("解析JSON数据失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"ec": 400,
			"em": "解析JSON数据失败",
		})
		return
	}

	// 验证基本数据格式
	ec, ok := data["ec"].(float64)
	if !ok || int(ec) != 200 {
		h.logger.Error("无效的数据格式", "ec", ec)
		c.JSON(http.StatusOK, gin.H{
			"ec": 400,
			"em": "无效的数据格式",
		})
		return
	}

	// 处理Webhook数据
	success, err := h.productService.ProcessAfdianWebhook(data)
	if err != nil {
		h.logger.Error("处理Webhook失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"ec": 500,
			"em": "处理订单失败: " + err.Error(),
		})
		return
	}

	if !success {
		c.JSON(http.StatusOK, gin.H{
			"ec": 400,
			"em": "订单处理失败",
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"ec": 200,
		"em": "",
	})
}

// GetUserOrders 获取用户的所有订单
func (h *ProductHandler) GetUserOrders(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"code":    401,
			"message": "用户未认证",
		})
		return
	}

	// 获取分页参数
	page := 1
	pageSize := 10

	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")

	if pageStr != "" {
		if pageNum, err := strconv.Atoi(pageStr); err == nil && pageNum > 0 {
			page = pageNum
		}
	}

	if pageSizeStr != "" {
		if pageSizeNum, err := strconv.Atoi(pageSizeStr); err == nil && pageSizeNum > 0 {
			pageSize = pageSizeNum
		}
	}

	// 将userID转换为uint64
	var userIDUint uint64
	switch v := userID.(type) {
	case int64:
		userIDUint = uint64(v)
	case uint64:
		userIDUint = v
	case float64:
		userIDUint = uint64(v)
	case int:
		userIDUint = uint64(v)
	default:
		h.logger.Error("无效的用户ID类型", "type", fmt.Sprintf("%T", userID), "value", userID)
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "服务器内部错误",
		})
		return
	}

	orders, total, err := h.productService.GetOrdersByUserID(userIDUint, page, pageSize)
	if err != nil {
		h.logger.Error("获取用户订单失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "获取用户订单失败",
		})
		return
	}

	// 计算总页数
	pages := (total + pageSize - 1) / pageSize
	if pages == 0 {
		pages = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"msg":    "获取成功",
		"orders": orders,
		"pagination": gin.H{
			"page":      page,
			"page_size": pageSize,
			"pages":     pages,
			"total":     total,
		},
	})
}

// GetOrderStatus 获取订单状态
func (h *ProductHandler) GetOrderStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"code":    401,
			"message": "用户未认证",
		})
		return
	}

	orderNo := c.Query("order_no")
	if orderNo == "" {
		c.JSON(http.StatusOK, gin.H{
			"code":    400,
			"message": "订单号不能为空",
		})
		return
	}

	order, err := h.productService.GetOrderByOrderNo(orderNo)
	if err != nil {
		h.logger.Error("获取订单状态失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "获取订单状态失败",
		})
		return
	}

	// 将userID转换为uint64
	var userIDUint uint64
	switch v := userID.(type) {
	case int64:
		userIDUint = uint64(v)
	case uint64:
		userIDUint = v
	case float64:
		userIDUint = uint64(v)
	case int:
		userIDUint = uint64(v)
	default:
		h.logger.Error("无效的用户ID类型", "type", fmt.Sprintf("%T", userID), "value", userID)
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "服务器内部错误",
		})
		return
	}

	// 验证订单所属用户
	if order.UserID != userIDUint {
		c.JSON(http.StatusOK, gin.H{
			"code":    403,
			"message": "无权访问此订单",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取订单状态成功",
		"data": gin.H{
			"order_no":        order.OrderNo,
			"status":          order.Status,
			"amount":          order.Amount,
			"product_name":    order.ProductName,
			"created_at":      order.CreatedAt,
			"paid_at":         order.PaidAt,
			"reward_executed": order.RewardExecuted,
		},
	})
}
