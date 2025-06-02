package admin

import (
	"context"
	"fmt"
	"net/http"
	"stellarfrp/internal/repository"
	"stellarfrp/internal/service"
	"stellarfrp/pkg/logger"

	"github.com/gin-gonic/gin"
)

// GroupAdminHandler 用户组管理处理器
type GroupAdminHandler struct {
	groupService service.GroupService
	logger       *logger.Logger
}

// NewGroupAdminHandler 创建用户组管理处理器实例
func NewGroupAdminHandler(groupService service.GroupService, logger *logger.Logger) *GroupAdminHandler {
	return &GroupAdminHandler{
		groupService: groupService,
		logger:       logger,
	}
}

// ListGroups 获取用户组列表
func (h *GroupAdminHandler) ListGroups(c *gin.Context) {
	groups, err := h.groupService.List(context.Background())
	if err != nil {
		h.logger.Error("获取用户组列表失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "获取用户组列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"msg":    "获取成功",
		"groups": groups,
	})
}

// GetGroup 获取单个用户组信息
func (h *GroupAdminHandler) GetGroup(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	_, err := fmt.Sscanf(idStr, "%d", &id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无效的用户组ID"})
		return
	}

	group, err := h.groupService.GetByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "用户组不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取成功",
		"data": group,
	})
}

// CreateGroupRequest 创建用户组请求
type CreateGroupRequest struct {
	Name              string `json:"name" binding:"required"`
	TunnelLimit       int    `json:"tunnel_limit" binding:"required"`
	BandwidthLimit    int    `json:"bandwidth_limit" binding:"required"`
	TrafficQuota      int64  `json:"traffic_quota" binding:"required"`
	CheckinMinTraffic int64  `json:"checkin_min_traffic"`
	CheckinMaxTraffic int64  `json:"checkin_max_traffic"`
}

// CreateGroup 创建用户组
func (h *GroupAdminHandler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("创建用户组参数绑定失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误：" + err.Error()})
		return
	}

	// 检查用户组名是否已存在
	existingGroup, err := h.groupService.GetByName(context.Background(), req.Name)
	if err == nil && existingGroup != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "用户组名已存在"})
		return
	}

	// 创建用户组
	group := &repository.Group{
		Name:              req.Name,
		TunnelLimit:       req.TunnelLimit,
		BandwidthLimit:    req.BandwidthLimit,
		TrafficQuota:      req.TrafficQuota,
		CheckinMinTraffic: req.CheckinMinTraffic,
		CheckinMaxTraffic: req.CheckinMaxTraffic,
	}

	err = h.groupService.Create(context.Background(), group)
	if err != nil {
		h.logger.Error("创建用户组失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "创建用户组失败：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "创建成功",
		"data": group,
	})
}

// UpdateGroupRequest 更新用户组请求
type UpdateGroupRequest struct {
	ID                int64   `json:"id" binding:"required"`
	Name              *string `json:"name"`
	TunnelLimit       *int    `json:"tunnel_limit"`
	BandwidthLimit    *int    `json:"bandwidth_limit"`
	TrafficQuota      *int64  `json:"traffic_quota"`
	CheckinMinTraffic *int64  `json:"checkin_min_traffic"`
	CheckinMaxTraffic *int64  `json:"checkin_max_traffic"`
}

// UpdateGroup 更新用户组
func (h *GroupAdminHandler) UpdateGroup(c *gin.Context) {
	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("更新用户组参数绑定失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误：" + err.Error()})
		return
	}

	// 获取现有用户组
	group, err := h.groupService.GetByID(context.Background(), req.ID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "用户组不存在"})
		return
	}

	// 更新用户组信息
	if req.Name != nil {
		// 检查新名称是否已存在（如果更改了名称）
		if *req.Name != group.Name {
			existingGroup, err := h.groupService.GetByName(context.Background(), *req.Name)
			if err == nil && existingGroup != nil {
				c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "用户组名已存在"})
				return
			}
		}
		group.Name = *req.Name
	}
	if req.TunnelLimit != nil {
		group.TunnelLimit = *req.TunnelLimit
	}
	if req.BandwidthLimit != nil {
		group.BandwidthLimit = *req.BandwidthLimit
	}
	if req.TrafficQuota != nil {
		group.TrafficQuota = *req.TrafficQuota
	}
	if req.CheckinMinTraffic != nil {
		group.CheckinMinTraffic = *req.CheckinMinTraffic
	}
	if req.CheckinMaxTraffic != nil {
		group.CheckinMaxTraffic = *req.CheckinMaxTraffic
	}

	// 保存更新
	err = h.groupService.Update(context.Background(), group)
	if err != nil {
		h.logger.Error("更新用户组失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "更新用户组失败：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "更新成功",
		"data": group,
	})
}

// DeleteGroupRequest 删除用户组请求
type DeleteGroupRequest struct {
	ID int64 `json:"id" binding:"required"`
}

// DeleteGroup 删除用户组
func (h *GroupAdminHandler) DeleteGroup(c *gin.Context) {
	var req DeleteGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("删除用户组参数绑定失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误：" + err.Error()})
		return
	}

	// 检查用户组是否存在
	_, err := h.groupService.GetByID(context.Background(), req.ID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "用户组不存在"})
		return
	}

	// 删除用户组
	err = h.groupService.Delete(context.Background(), req.ID)
	if err != nil {
		h.logger.Error("删除用户组失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "删除用户组失败：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "删除成功",
	})
}

// SearchGroups 搜索用户组
func (h *GroupAdminHandler) SearchGroups(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "搜索关键词不能为空"})
		return
	}

	groups, err := h.groupService.SearchGroups(context.Background(), keyword)
	if err != nil {
		h.logger.Error("搜索用户组失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "搜索用户组失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"msg":    "搜索成功",
		"groups": groups,
	})
}
