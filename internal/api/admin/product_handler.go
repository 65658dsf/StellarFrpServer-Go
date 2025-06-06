package admin

import (
	"fmt"
	"net/http"
	"stellarfrp/internal/model"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ProductAdminHandler 管理员商品处理器
type ProductAdminHandler struct {
	productService *service.ProductService
	userService    service.UserService
	logger         *logger.Logger
}

// NewProductAdminHandler 创建管理员商品处理器
func NewProductAdminHandler(productService *service.ProductService, userService service.UserService, logger *logger.Logger) *ProductAdminHandler {
	return &ProductAdminHandler{
		productService: productService,
		userService:    userService,
		logger:         logger,
	}
}

// ListProducts 获取所有商品列表
func (h *ProductAdminHandler) ListProducts(c *gin.Context) {
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

	products, total, err := h.productService.GetProductsWithPagination(page, pageSize)
	if err != nil {
		h.logger.Error("获取商品列表失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "获取商品列表失败",
		})
		return
	}

	// 计算总页数
	pages := (total + pageSize - 1) / pageSize
	if pages == 0 {
		pages = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"code":     200,
		"msg":      "获取商品列表成功",
		"products": products,
		"pagination": gin.H{
			"page":      page,
			"page_size": pageSize,
			"pages":     pages,
			"total":     total,
		},
	})
}

// GetProduct 获取单个商品信息
func (h *ProductAdminHandler) GetProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "无效的商品ID",
		})
		return
	}

	product, err := h.productService.GetProductByID(id)
	if err != nil {
		h.logger.Error("获取商品信息失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "获取商品信息失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"msg":     "获取商品信息成功",
		"product": product,
	})
}

// CreateProduct 创建商品
func (h *ProductAdminHandler) CreateProduct(c *gin.Context) {
	var product model.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "无效的请求参数",
		})
		return
	}

	if err := h.productService.CreateProduct(&product); err != nil {
		h.logger.Error("创建商品失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "创建商品失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"msg":     "创建商品成功",
		"product": product,
	})
}

// UpdateProduct 更新商品
func (h *ProductAdminHandler) UpdateProduct(c *gin.Context) {
	var product model.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "无效的请求参数",
		})
		return
	}

	if err := h.productService.UpdateProduct(&product); err != nil {
		h.logger.Error("更新商品失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "更新商品失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"msg":     "更新商品成功",
		"product": product,
	})
}

// DeleteProduct 删除商品
func (h *ProductAdminHandler) DeleteProduct(c *gin.Context) {
	var req struct {
		ID uint64 `json:"id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "无效的请求参数",
		})
		return
	}

	if err := h.productService.DeleteProduct(req.ID); err != nil {
		h.logger.Error("删除商品失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "删除商品失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "删除商品成功",
	})
}

// ListOrders 获取所有订单列表（带分页）
func (h *ProductAdminHandler) ListOrders(c *gin.Context) {
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

	// 获取可选的过滤参数
	userIDStr := c.Query("user_id")
	statusStr := c.Query("status")
	orderNo := c.Query("order_no")

	var userID *uint64
	var status *int

	if userIDStr != "" {
		if uid, err := strconv.ParseUint(userIDStr, 10, 64); err == nil {
			userID = &uid
		}
	}

	if statusStr != "" {
		if s, err := strconv.Atoi(statusStr); err == nil {
			status = &s
		}
	}

	orders, total, err := h.productService.GetOrdersWithFilter(page, pageSize, userID, status, orderNo)
	if err != nil {
		h.logger.Error("获取订单列表失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "获取订单列表失败",
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
		"msg":    "获取订单列表成功",
		"orders": orders,
		"pagination": gin.H{
			"page":      page,
			"page_size": pageSize,
			"pages":     pages,
			"total":     total,
		},
	})
}

// GetOrder 获取单个订单信息
func (h *ProductAdminHandler) GetOrder(c *gin.Context) {
	orderNo := c.Param("order_no")
	if orderNo == "" {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "订单号不能为空",
		})
		return
	}

	order, err := h.productService.GetOrderByOrderNo(orderNo)
	if err != nil {
		h.logger.Error("获取订单信息失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "获取订单信息失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":  200,
		"msg":   "获取订单信息成功",
		"order": order,
	})
}

// UpdateOrderStatus 更新订单状态
func (h *ProductAdminHandler) UpdateOrderStatus(c *gin.Context) {
	var req struct {
		OrderNo string `json:"order_no" binding:"required"`
		Status  int    `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "无效的请求参数",
		})
		return
	}

	// 获取当前订单信息
	order, err := h.productService.GetOrderByOrderNo(req.OrderNo)
	if err != nil {
		h.logger.Error("获取订单信息失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "获取订单信息失败",
		})
		return
	}

	if order == nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 404,
			"msg":  "订单不存在",
		})
		return
	}

	// 更新订单状态
	if err := h.productService.UpdateOrderStatus(req.OrderNo, req.Status, ""); err != nil {
		h.logger.Error("更新订单状态失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "更新订单状态失败",
		})
		return
	}

	// 如果订单状态改为已支付，且之前未支付，则执行奖励
	if req.Status == 1 && order.Status != 1 {
		// 设置支付时间
		if err := h.productService.UpdateOrderPaidTime(req.OrderNo, time.Now()); err != nil {
			h.logger.Error("更新订单支付时间失败", "error", err)
		}

		// 执行奖励
		if err := h.productService.ExecuteOrderReward(req.OrderNo); err != nil {
			h.logger.Error("执行订单奖励失败", "error", err)
			c.JSON(http.StatusOK, gin.H{
				"code": 500,
				"msg":  fmt.Sprintf("订单状态已更新，但执行奖励失败: %s", err.Error()),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "更新订单状态成功",
	})
}

// DeleteOrder 删除订单
func (h *ProductAdminHandler) DeleteOrder(c *gin.Context) {
	var req struct {
		OrderNo string `json:"order_no" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "无效的请求参数",
		})
		return
	}

	if err := h.productService.DeleteOrder(req.OrderNo); err != nil {
		h.logger.Error("删除订单失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "删除订单失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "删除订单成功",
	})
}
