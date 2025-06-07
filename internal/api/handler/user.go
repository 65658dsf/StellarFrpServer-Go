package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"stellarfrp/internal/constants"
	"stellarfrp/internal/repository"
	"stellarfrp/internal/service"
	"stellarfrp/internal/types"
	"stellarfrp/pkg/email"
	"stellarfrp/pkg/geetest"
	"stellarfrp/pkg/logger"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/util/rand"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService   service.UserService
	redisClient   *redis.Client
	emailService  *email.Service
	logger        *logger.Logger
	geetestClient *geetest.GeetestClient
	proxyService  service.ProxyService
}

// NewUserHandler 创建用户处理器实例
func NewUserHandler(userService service.UserService, redisClient *redis.Client, emailService *email.Service, logger *logger.Logger, geetestClient *geetest.GeetestClient, proxyService service.ProxyService) *UserHandler {
	return &UserHandler{
		userService:   userService,
		redisClient:   redisClient,
		emailService:  emailService,
		logger:        logger,
		geetestClient: geetestClient,
		proxyService:  proxyService,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

// Register 用户注册
func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误或参数不足"})
		return
	}

	// 验证邮箱验证码
	codeKey := "email_verify:" + req.Email
	code, err := h.redisClient.Get(context.Background(), codeKey).Result()
	if err == redis.Nil || code != req.Code {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "验证码错误或已过期"})
		return
	}

	// 使用分布式锁控制并发
	lockKey := "user_register:" + req.Email
	lock := h.redisClient.SetNX(context.Background(), lockKey, "1", 10*time.Second)
	if !lock.Val() {
		c.JSON(http.StatusOK, gin.H{"code": 429, "msg": "请求过于频繁，请稍后重试"})
		return
	}
	defer h.redisClient.Del(context.Background(), lockKey)

	// 验证用户名格式
	if !regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`).MatchString(req.Username) {
		c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "用户名只能为英文或数字"})
		return
	}

	// 验证密码格式
	if !regexp.MustCompile(`^[A-Za-z0-9]{6,}$`).MatchString(req.Password) || !regexp.MustCompile(`[A-Za-z]`).MatchString(req.Password) || !regexp.MustCompile(`[0-9]`).MatchString(req.Password) {
		c.JSON(http.StatusOK, gin.H{"code": 404, "msg": "密码必须为英文加数字长度在6位以上的密码"})
		return
	}

	// 验证邮箱格式
	if !regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@(qq\.com|163\.com|outlook\.com|gmail\.com)$`).MatchString(req.Email) {
		c.JSON(http.StatusOK, gin.H{"code": 405, "msg": "邮箱格式不正确"})
		return
	}

	// 检查用户名是否已存在
	if _, err := h.userService.GetByUsername(context.Background(), req.Username); err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 406, "msg": "用户名重复"})
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
		Username:    req.Username,
		Password:    string(hashedPassword),
		Email:       req.Email,
		Status:      1, // 正常状态
		GroupID:     1, // 默认用户组
		IsVerified:  0, // 未实名认证
		VerifyInfo:  sql.NullString{String: "", Valid: true},
		VerifyCount: 0,
		Token:       rand.String(32), // 生成随机Token
	}

	if err := h.userService.Create(context.Background(), user); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}

	// 删除验证码
	h.redisClient.Del(context.Background(), codeKey)

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "用户已成功注册"})
}

// Create 创建用户
func (h *UserHandler) Create(c *gin.Context) {
	var user repository.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	if err := h.userService.Create(context.Background(), &user); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "创建成功"})
}

// CreateAsync 异步创建用户
func (h *UserHandler) CreateAsync(c *gin.Context) {
	var user repository.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	taskID, err := h.userService.CreateAsync(context.Background(), &user)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "任务已提交", "data": gin.H{"task_id": taskID}})
}

// GetTaskStatus 获取任务状态
func (h *UserHandler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("id")
	status, err := h.userService.GetTaskStatus(context.Background(), taskID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "获取成功", "data": gin.H{"status": status}})
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"account" binding:"required"` // 用户名或邮箱
	Password string `json:"password" binding:"required"`
}

// Login 用户登录
func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": constants.ErrInvalidParams})
		return
	}

	// 调用登录服务
	user, err := h.userService.Login(context.Background(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": err.Error()})
		return
	}

	// 只返回token
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  constants.SuccessLogin,
		"data": gin.H{
			"token": user.Token,
		},
	})
}

// GetByID 根据ID获取用户
func (h *UserHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	user, err := h.userService.GetByID(context.Background(), idInt)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "获取成功", "data": user})
}

// List 获取用户列表
func (h *UserHandler) List(c *gin.Context) {
	page := 1
	pageSize := 10

	users, err := h.userService.List(context.Background(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "获取成功", "data": users})
}

// Update 更新用户
func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	var user repository.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	user.ID = idInt
	if err := h.userService.Update(context.Background(), &user); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "更新成功"})
}

// Delete 删除用户
func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	if err := h.userService.Delete(context.Background(), idInt); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "删除成功"})
}

// GetUserInfo 获取用户信息
func (h *UserHandler) GetUserInfo(c *gin.Context) {
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

	// 确定实际使用的用户组ID
	effectiveGroupID := user.GroupID
	if user.IsVerified == 0 && user.GroupID != 6 {
		effectiveGroupID = 1 // 未实名用户组ID为1
	}

	// 获取用户组名称
	groupName, err := h.userService.GetGroupName(context.Background(), effectiveGroupID)
	if err != nil {
		groupName = "未知用户组"
		h.logger.Error("Failed to get group name", "error", err)
	}

	// 获取用户组流量
	groupTraffic, err := h.userService.GetGroupTraffic(context.Background(), effectiveGroupID)
	if err != nil {
		h.logger.Error("Failed to get group traffic", "error", err)
		groupTraffic = 0
	}

	// 获取用户自身的额外流量
	userTraffic := int64(0)
	if user.TrafficQuota != nil {
		userTraffic = *user.TrafficQuota
	}

	// 计算总流量
	totalTraffic := groupTraffic + userTraffic

	// 获取用户已使用的流量
	usedTraffic, err := h.userService.GetUserUsedTraffic(context.Background(), user.ID)
	if err != nil {
		h.logger.Error("Failed to get user used traffic", "error", err)
		usedTraffic = 0
	}

	// 获取用户当前的隧道数量
	proxyCount, err := h.proxyService.GetUserProxyCount(context.Background(), user.Username)
	if err != nil {
		h.logger.Error("Failed to get user proxy count", "error", err)
		proxyCount = 0
	}

	// 获取用户组的隧道数量限制
	tunnelLimit, err := h.userService.GetGroupTunnelLimit(context.Background(), effectiveGroupID)
	if err != nil {
		h.logger.Error("Failed to get group tunnel limit", "error", err)
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
		h.logger.Error("Failed to get user group for bandwidth", "error", err)
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

	userInfo := gin.H{
		"ID":           user.ID,
		"Username":     user.Username,
		"Email":        user.Email,
		"RegisterTime": user.RegisterTime.Format("2006-01-02 15:04:05"),
		"GroupID":      user.GroupID,
		"GroupName":    groupName,
		"IsVerified":   user.IsVerified,
		"Status":       user.Status,
		"VerifyCount":  user.VerifyCount,
		"Traffic":      totalTraffic,                                       // 总流量信息
		"UsedTraffic":  usedTraffic,                                        // 已使用流量信息
		"Tunnels":      fmt.Sprintf("%d/%d", proxyCount, totalTunnelLimit), // 隧道数量：已用/总数
		"Bandwidth":    totalBandwidth,                                     // 带宽限制(Mbps)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "获取成功", "data": userInfo})
}

// SendMessage 发送验证码
func (h *UserHandler) SendMessage(c *gin.Context) {
	var req types.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 验证邮箱格式
	if !regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@(qq\.com|163\.com|outlook\.com|gmail\.com)$`).MatchString(req.Email) {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "邮箱格式不正确"})
		return
	}

	// 检查极验验证配置是否有效
	if h.geetestClient.CaptchaID == "" || h.geetestClient.CaptchaKey == "" || h.geetestClient.APIServer == "" {
		h.logger.Error("极验验证配置无效", "captcha_id", h.geetestClient.CaptchaID, "api_server", h.geetestClient.APIServer)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器配置错误"})
		return
	}

	// 验证极验验证码
	// 优先使用validate对象中的参数
	if req.Validate != nil {
		// 构建验证参数
		verifyParams := geetest.VerifyParams{
			LotNumber:     req.Validate.LotNumber,
			CaptchaOutput: req.Validate.CaptchaOutput,
			PassToken:     req.Validate.PassToken,
			GenTime:       req.Validate.GenTime,
		}

		// 验证极验验证码
		verified, err := h.geetestClient.Verify(verifyParams)
		if err != nil || !verified {
			errorMsg := "人机验证失败"
			if err != nil {
				errorMsg = err.Error()
			}
			h.logger.Error("人机验证失败", "error", err, "lot_number", req.Validate.LotNumber)
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": errorMsg})
			return
		}
	} else if req.LotNumber != "" {
		// 兼容旧版参数
		verifyParams := geetest.VerifyParams{
			LotNumber:     req.LotNumber,
			CaptchaOutput: req.CaptchaOutput,
			PassToken:     req.PassToken,
			GenTime:       req.GenTime,
		}

		// 验证极验验证码
		verified, err := h.geetestClient.Verify(verifyParams)
		if err != nil || !verified {
			errorMsg := "人机验证失败"
			if err != nil {
				errorMsg = err.Error()
			}
			h.logger.Error("人机验证失败", "error", err, "lot_number", req.LotNumber)
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": errorMsg})
			return
		}
	} else {
		// 没有提供验证参数
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "请完成人机验证"})
		return
	}

	// 使用分布式锁控制发送频率
	lockKey := "send_code:" + req.Email
	lock := h.redisClient.SetNX(context.Background(), lockKey, "1", time.Minute)
	if !lock.Val() {
		c.JSON(http.StatusOK, gin.H{"code": 429, "msg": "发送过于频繁，请稍后重试"})
		return
	}

	// 发送验证码邮件
	if err := h.userService.SendEmail(context.Background(), req.Email, req.Type); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "发送验证码失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "验证码已发送"})
}

// ResetTokenRequest 重置Token请求
type ResetTokenRequest struct {
	Account  string `json:"account" binding:"required"`  // 用户名或邮箱
	Password string `json:"password" binding:"required"` // 密码
}

// ResetToken 重置用户Token
func (h *UserHandler) ResetToken(c *gin.Context) {
	var req ResetTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	// 调用重置Token服务
	user, err := h.userService.ResetToken(context.Background(), req.Account, req.Password)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": err.Error()})
		return
	}

	// 返回新的token
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "Token重置成功",
		"data": gin.H{
			"token": user.Token,
		},
	})
}

// GetGroupList 获取权限组列表
func (h *UserHandler) GetGroupList(c *gin.Context) {
	// 获取所有用户组
	groups, err := h.userService.GetAllGroups(context.Background())
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
