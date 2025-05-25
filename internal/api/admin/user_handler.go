package admin

import (
	"context"
	"database/sql"
	"net/http"
	"stellarfrp/internal/repository"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// UserAdminHandler 用户管理处理器
type UserAdminHandler struct {
	userService service.UserService
	logger      *logger.Logger
}

// NewUserAdminHandler 创建用户管理处理器实例
func NewUserAdminHandler(userService service.UserService, logger *logger.Logger) *UserAdminHandler {
	return &UserAdminHandler{
		userService: userService,
		logger:      logger,
	}
}

// ListUsers 获取用户列表
func (h *UserAdminHandler) ListUsers(c *gin.Context) {
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

	// 获取用户列表
	users, err := h.userService.List(context.Background(), page, pageSize)
	if err != nil {
		h.logger.Error("获取用户列表失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取用户列表失败"})
		return
	}

	// 获取总用户数
	total, err := h.userService.Count(context.Background())
	if err != nil {
		h.logger.Error("获取用户总数失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取用户总数失败"})
		return
	}

	// 计算总页数
	pages := (total + int64(pageSize) - 1) / int64(pageSize)

	// 用户列表增强信息
	type EnhancedUser struct {
		*repository.User
		GroupName        string `json:"GroupName"`
		TotalTunnelLimit int    `json:"GroupTunnel"`
		TotalBandwidth   int    `json:"GroupBandwidth"`
	}

	// 增强用户信息
	enhancedUsers := make([]EnhancedUser, 0, len(users))
	for _, user := range users {
		// 移除敏感信息
		user.Password = ""
		user.Token = ""

		// 获取用户组名称
		groupName, err := h.userService.GetGroupName(context.Background(), user.GroupID)
		if err != nil {
			groupName = "未知用户组"
		}

		// 获取用户组的隧道数量限制
		tunnelLimit, err := h.userService.GetGroupTunnelLimit(context.Background(), user.GroupID)
		if err != nil {
			h.logger.Error("获取用户组隧道限制失败", "error", err, "user_id", user.ID)
			tunnelLimit = 0
		}

		// 获取用户组带宽限制
		userGroup, err := h.userService.GetUserGroup(context.Background(), user.ID)
		if err != nil {
			h.logger.Error("获取用户组信息失败", "error", err, "user_id", user.ID)
		}

		enhancedUsers = append(enhancedUsers, EnhancedUser{
			User:             user,
			GroupName:        groupName,
			TotalTunnelLimit: tunnelLimit,
			TotalBandwidth:   userGroup.BandwidthLimit,
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
		"users": enhancedUsers,
	})
}

// GetUser 获取单个用户信息
func (h *UserAdminHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	user, err := h.userService.GetByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "用户不存在"})
		return
	}

	// 移除敏感信息
	user.Password = ""
	user.Token = ""

	// 获取用户组名称
	groupName, err := h.userService.GetGroupName(context.Background(), user.GroupID)
	if err != nil {
		groupName = "未知用户组"
	}

	// 获取用户组的隧道数量限制
	tunnelLimit, err := h.userService.GetGroupTunnelLimit(context.Background(), user.GroupID)
	if err != nil {
		h.logger.Error("获取用户组隧道限制失败", "error", err, "user_id", user.ID)
		tunnelLimit = 0
	}

	// 获取用户的附加隧道数量
	additionalTunnels := 0
	if user.TunnelCount != nil {
		additionalTunnels = *user.TunnelCount
	}

	// 计算总的隧道数量限制
	totalTunnelLimit := tunnelLimit + additionalTunnels

	// 获取用户组带宽限制
	userGroup, err := h.userService.GetUserGroup(context.Background(), user.ID)
	if err != nil {
		h.logger.Error("获取用户组信息失败", "error", err, "user_id", user.ID)
	}

	// 计算总带宽限制
	userBandwidth := 0
	if user.Bandwidth != nil {
		userBandwidth = *user.Bandwidth
	}

	totalBandwidth := 0
	if userGroup != nil {
		totalBandwidth = userGroup.BandwidthLimit + userBandwidth
	}

	// 用户信息增强
	enhancedUser := struct {
		*repository.User
		GroupName        string `json:"GroupName"`
		TotalTunnelLimit int    `json:"GroupTunnel"`
		TotalBandwidth   int    `json:"GroupBandwidth"`
	}{
		User:             user,
		GroupName:        groupName,
		TotalTunnelLimit: totalTunnelLimit,
		TotalBandwidth:   totalBandwidth,
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取成功",
		"data": enhancedUser,
	})
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username     string  `json:"username" binding:"required"`
	Password     string  `json:"password" binding:"required"`
	Email        string  `json:"email" binding:"required"`
	GroupID      int64   `json:"group_id" binding:"required"`
	Status       int     `json:"status" binding:"required"`
	TunnelCount  *int    `json:"tunnel_count"`
	Bandwidth    *int    `json:"bandwidth"`
	TrafficQuota *int64  `json:"traffic_quota"`
	GroupTime    *string `json:"group_time"`
}

// CreateUser 创建用户
func (h *UserAdminHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 检查用户名是否已存在
	if _, err := h.userService.GetByUsername(context.Background(), req.Username); err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 409, "msg": "用户名已存在"})
		return
	}

	// 检查邮箱是否已被注册
	if _, err := h.userService.GetByEmail(context.Background(), req.Email); err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 409, "msg": "该邮箱已被注册"})
		return
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	// 创建用户
	user := &repository.User{
		Username:     req.Username,
		Password:     string(hashedPassword),
		Email:        req.Email,
		Status:       req.Status,
		GroupID:      req.GroupID,
		IsVerified:   0, // 默认未实名认证
		VerifyInfo:   sql.NullString{String: "", Valid: true},
		VerifyCount:  0,
		TunnelCount:  req.TunnelCount,
		Bandwidth:    req.Bandwidth,
		TrafficQuota: req.TrafficQuota,
	}

	if err := h.userService.Create(context.Background(), user); err != nil {
		h.logger.Error("创建用户失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "创建用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "创建成功", "data": gin.H{"id": user.ID}})
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Username     string  `json:"username"`
	Password     string  `json:"password"`
	Email        string  `json:"email"`
	GroupID      *int64  `json:"group_id"`
	Status       *int    `json:"status"`
	TunnelCount  *int    `json:"tunnel_count"`
	Bandwidth    *int    `json:"bandwidth"`
	TrafficQuota *int64  `json:"traffic_quota"`
	GroupTime    *string `json:"group_time"`
}

// UpdateUser 更新用户
func (h *UserAdminHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 获取用户
	user, err := h.userService.GetByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "用户不存在"})
		return
	}

	// 更新用户信息
	if req.Username != "" {
		// 检查用户名是否已存在
		existingUser, err := h.userService.GetByUsername(context.Background(), req.Username)
		if err == nil && existingUser.ID != id {
			c.JSON(http.StatusOK, gin.H{"code": 409, "msg": "用户名已存在"})
			return
		}
		user.Username = req.Username
	}

	if req.Email != "" {
		// 检查邮箱是否已被注册
		existingUser, err := h.userService.GetByEmail(context.Background(), req.Email)
		if err == nil && existingUser.ID != id {
			c.JSON(http.StatusOK, gin.H{"code": 409, "msg": "该邮箱已被注册"})
			return
		}
		user.Email = req.Email
	}

	if req.Password != "" {
		// 密码加密
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
			return
		}
		user.Password = string(hashedPassword)
	}

	if req.GroupID != nil {
		user.GroupID = *req.GroupID
	}

	if req.Status != nil {
		user.Status = *req.Status
	}

	if req.TunnelCount != nil {
		user.TunnelCount = req.TunnelCount
	}

	if req.Bandwidth != nil {
		user.Bandwidth = req.Bandwidth
	}

	if req.TrafficQuota != nil {
		user.TrafficQuota = req.TrafficQuota
	}

	if err := h.userService.Update(context.Background(), user); err != nil {
		h.logger.Error("更新用户失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "更新用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "更新成功"})
}

// DeleteUser 删除用户
func (h *UserAdminHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	if err := h.userService.Delete(context.Background(), id); err != nil {
		h.logger.Error("删除用户失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "删除用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "删除成功"})
}

// SearchUsers 搜索用户
func (h *UserAdminHandler) SearchUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "搜索关键字不能为空"})
		return
	}

	// 这里需要在service层添加搜索方法，暂时返回空结果
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "搜索成功",
		"data": []interface{}{},
	})
}

// ResetUserToken 重置用户Token
func (h *UserAdminHandler) ResetUserToken(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	// 获取用户
	user, err := h.userService.GetByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "用户不存在"})
		return
	}

	// 重置Token
	if err := h.userService.AdminResetToken(context.Background(), user); err != nil {
		h.logger.Error("重置Token失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "重置Token失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "重置成功",
		"data": gin.H{"token": user.Token},
	})
}
