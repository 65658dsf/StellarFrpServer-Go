package service

import (
	"context"
	"errors"
	"fmt"
	"stellarfrp/internal/constants"
	"stellarfrp/internal/repository"
	"stellarfrp/pkg/async"
	"stellarfrp/pkg/email"
	"stellarfrp/pkg/logger"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/util/rand"
)

// const (
// 	allUsersCacheDuration = 5 * time.Minute // 不再使用
// )

// const allUsersCacheKey = "users:all" // 不再使用

// UserService 用户服务接口
type UserService interface {
	Create(ctx context.Context, user *repository.User) error
	CreateAsync(ctx context.Context, user *repository.User) (string, error)
	GetByID(ctx context.Context, id int64) (*repository.User, error)
	GetByUsername(ctx context.Context, username string) (*repository.User, error)
	GetByEmail(ctx context.Context, email string) (*repository.User, error)
	GetByToken(ctx context.Context, token string) (*repository.User, error)
	Update(ctx context.Context, user *repository.User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, page, pageSize int) ([]*repository.User, error)
	GetAllUsers(ctx context.Context) ([]*repository.User, error)
	GetUsersWithExpiredGroup(ctx context.Context, groupID int64) ([]*repository.User, error)
	Count(ctx context.Context) (int64, error)
	GetTaskStatus(ctx context.Context, taskID string) (string, error)
	SendEmail(ctx context.Context, email, msgType string) error
	Login(ctx context.Context, identifier, password string) (*repository.User, error)
	GetGroupName(ctx context.Context, groupID int64) (string, error)
	GetGroupTunnelLimit(ctx context.Context, groupID int64) (int, error)
	GetUserBandwidth(ctx context.Context, userID int64) (int, error)
	GetUserTrafficQuota(ctx context.Context, userID int64) (int64, error)
	GetUserGroup(ctx context.Context, userID int64) (*repository.Group, error)
	ResetToken(ctx context.Context, identifier, password string) (*repository.User, error)
	AdminResetToken(ctx context.Context, user *repository.User) error
	GetGroupTraffic(ctx context.Context, groupID int64) (int64, error)
	GetUserUsedTraffic(ctx context.Context, userID int64) (int64, error)
	IsUserBlacklisted(ctx context.Context, userID int64) (bool, error)
	IsUserBlacklistedByUsername(ctx context.Context, username string) (bool, error)
	IsUserBlacklistedByToken(ctx context.Context, token string) (bool, error)
	SearchUsers(ctx context.Context, keyword string) ([]*repository.User, error)
}

// userService 用户服务实现
type userService struct {
	userRepo        repository.UserRepository
	groupRepo       repository.GroupRepository
	userTrafficRepo repository.UserTrafficLogRepository
	redisClient     *redis.Client
	emailSvc        *email.Service
	logger          *logger.Logger
	worker          *async.Worker
}

// NewUserService 创建用户服务实例
func NewUserService(
	userRepo repository.UserRepository,
	groupRepo repository.GroupRepository,
	userTrafficRepo repository.UserTrafficLogRepository,
	redisClient *redis.Client,
	worker *async.Worker,
	emailSvc *email.Service,
	logger *logger.Logger,
) UserService {
	return &userService{
		userRepo:        userRepo,
		groupRepo:       groupRepo,
		userTrafficRepo: userTrafficRepo,
		redisClient:     redisClient,
		worker:          worker,
		emailSvc:        emailSvc,
		logger:          logger,
	}
}

// Create 创建用户
func (s *userService) Create(ctx context.Context, user *repository.User) error {
	// 检查用户名是否已存在
	existUser, err := s.userRepo.GetByUsername(ctx, user.Username)
	if err == nil && existUser != nil {
		return errors.New("username already exists")
	}

	// 检查邮箱是否已存在
	existUser, err = s.userRepo.GetByEmail(ctx, user.Email)
	if err == nil && existUser != nil {
		return errors.New("email already exists")
	}

	// 创建用户
	if err := s.userRepo.Create(ctx, user); err != nil {
		return err
	}

	// s.redisClient.Del(ctx, allUsersCacheKey) // 不再使用 allUsersCacheKey

	// 发送欢迎邮件
	go s.emailSvc.SendWelcomeEmail(user.Email, user.Username)

	return nil
}

// CreateAsync 异步创建用户
func (s *userService) CreateAsync(ctx context.Context, user *repository.User) (string, error) {
	taskID := fmt.Sprintf("create_user_%d", time.Now().UnixNano())

	// 添加异步任务
	s.worker.AddTask(func() {
		if err := s.Create(context.Background(), user); err != nil {
			s.logger.Error("Failed to create user", "error", err)
			s.redisClient.Set(ctx, taskID, "failed", 24*time.Hour)
			return
		}
		s.redisClient.Set(ctx, taskID, "success", 24*time.Hour)
	})

	// 设置初始状态
	s.redisClient.Set(ctx, taskID, "processing", 24*time.Hour)

	return taskID, nil
}

// GetTaskStatus 获取任务状态
func (s *userService) GetTaskStatus(ctx context.Context, taskID string) (string, error) {
	status, err := s.redisClient.Get(ctx, taskID).Result()
	if err == redis.Nil {
		return "", errors.New("task not found")
	}
	if err != nil {
		return "", err
	}
	return status, nil
}

// GetByID 根据ID获取用户
func (s *userService) GetByID(ctx context.Context, id int64) (*repository.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetAllUsers 获取所有用户
func (s *userService) GetAllUsers(ctx context.Context) ([]*repository.User, error) {
	// cachedData, err := s.redisClient.Get(ctx, allUsersCacheKey).Result() // 不再使用缓存
	// if err == nil {
	// 	var users []*repository.User
	// 	if json.Unmarshal([]byte(cachedData), &users) == nil {
	// 		return users, nil
	// 	}
	// 	s.logger.Error("Failed to unmarshal cached all users data", "error", err)
	// } else if err != redis.Nil {
	// 	s.logger.Error("Failed to get all users from cache", "error", err)
	// }

	// 数据库分页获取所有用户，避免一次性加载过多数据到内存
	var allUsers []*repository.User
	pageSize := 100 // 根据实际情况调整
	for page := 1; ; page++ {
		users, listErr := s.userRepo.List(ctx, (page-1)*pageSize, pageSize)
		if listErr != nil {
			s.logger.Error("Failed to list users from repository", "error", listErr, "page", page)
			// 如果是第一页就出错，则返回错误，否则尝试返回已获取的部分
			if page == 1 {
				return nil, listErr
			}
			break
		}
		if len(users) == 0 {
			break // 没有更多用户了
		}
		allUsers = append(allUsers, users...)
		if len(users) < pageSize {
			break // 最后一页
		}
	}

	// if len(allUsers) > 0 { // 不再写入缓存
	// 	jsonData, marshalErr := json.Marshal(allUsers)
	// 	if marshalErr == nil {
	// 		setErr := s.redisClient.Set(ctx, allUsersCacheKey, jsonData, allUsersCacheDuration).Err()
	// 		if setErr != nil {
	// 			s.logger.Error("Failed to set all users to cache", "error", setErr)
	// 		}
	// 	} else {
	// 		s.logger.Error("Failed to marshal all users data for cache", "error", marshalErr)
	// 	}
	// }
	return allUsers, nil
}

// GetUsersWithExpiredGroup 获取指定组中权限已过期的用户
func (s *userService) GetUsersWithExpiredGroup(ctx context.Context, groupID int64) ([]*repository.User, error) {
	// 注意：此方法通常不建议缓存，因为它依赖于当前时间，且结果集可能经常变化。
	// 如果确实需要缓存，缓存时间应较短，并且需要有策略在用户组信息变更时清除缓存。
	return s.userRepo.GetExpiredUsersByGroupID(ctx, groupID, time.Now())
}

// Update 更新用户信息
func (s *userService) Update(ctx context.Context, user *repository.User) error {
	err := s.userRepo.Update(ctx, user)
	if err == nil {
		// 更新成功，使相关缓存失效
		// s.redisClient.Del(ctx, allUsersCacheKey) // 不再使用 allUsersCacheKey
	}
	return err
}

// Delete 删除用户
func (s *userService) Delete(ctx context.Context, id int64) error {
	err := s.userRepo.Delete(ctx, id)
	if err == nil {
		// 删除成功，使相关缓存失效
		// s.redisClient.Del(ctx, allUsersCacheKey) // 不再使用 allUsersCacheKey
	}
	return err
}

// List 获取用户列表
func (s *userService) List(ctx context.Context, page, pageSize int) ([]*repository.User, error) {
	offset := (page - 1) * pageSize
	return s.userRepo.List(ctx, offset, pageSize)
}

// SendEmail 发送邮件
func (s *userService) SendEmail(ctx context.Context, email, msgType string) error {
	// 生成6位随机验证码
	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	// 根据消息类型发送不同的邮件
	var err error
	switch msgType {
	case "register":
		err = s.emailSvc.SendVerificationCode(email, code, 5)
	case "reset_password":
		// 获取用户信息，如果找不到用户则使用邮箱作为用户名
		user, userErr := s.userRepo.GetByEmail(ctx, email) // 直接从repo获取
		userName := email
		if userErr == nil && user != nil {
			userName = user.Username
		}
		err = s.emailSvc.SendPasswordResetCode(email, userName, code, 5)
	default:
		return fmt.Errorf("unsupported message type: %s", msgType)
	}

	if err != nil {
		s.logger.Error("Failed to send email", "error", err)
		return err
	}

	// 将验证码保存到Redis，设置5分钟过期
	key := "email_verify:" + email
	err = s.redisClient.Set(ctx, key, code, 5*time.Minute).Err()
	if err != nil {
		s.logger.Error("Failed to save verification code", "error", err)
		return err
	}

	return nil
}

// GetByUsername 根据用户名获取用户
func (s *userService) GetByUsername(ctx context.Context, username string) (*repository.User, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetByEmail 根据邮箱获取用户
func (s *userService) GetByEmail(ctx context.Context, email string) (*repository.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// Login 用户登录
func (s *userService) Login(ctx context.Context, identifier, password string) (*repository.User, error) {
	// 尝试通过用户名或邮箱获取用户
	var user *repository.User
	var err error

	// 先尝试用户名登录
	user, err = s.GetByUsername(ctx, identifier)
	if errors.Is(err, redis.Nil) || (err != nil && err.Error() == "用户不存在") {
		// 如果用户名不存在，尝试邮箱登录
		user, err = s.userRepo.GetByEmail(ctx, identifier)
		if err != nil {
			return nil, errors.New(constants.ErrUserNotFound)
		}
	} else if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New(constants.ErrUserNotFound)
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New(constants.ErrPasswordIncorrect)
	}

	// 只有在用户没有token或token为空时才生成新的token
	if user.Token == "" {
		// 生成固定的32位随机token
		user.Token = rand.String(32)

		// 更新用户token
		if err := s.Update(ctx, user); err != nil {
			return nil, err
		}
	}

	return user, nil
}

// GetGroupName 根据组ID获取组名称
func (s *userService) GetGroupName(ctx context.Context, groupID int64) (string, error) {
	group, err := s.getCachedGroupByID(ctx, groupID)
	if err != nil {
		s.logger.Error("Failed to get group", "error", err, "groupID", groupID)
		return "未知用户组", err
	}
	if group == nil {
		s.logger.Warn("Group not found for GetGroupName", "groupID", groupID)
		return "未知用户组", errors.New("group not found")
	}
	return group.Name, nil
}

// GetByToken 根据Token获取用户
func (s *userService) GetByToken(ctx context.Context, token string) (*repository.User, error) {
	user, err := s.userRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetGroupTunnelLimit 获取用户组的隧道数量限制
func (s *userService) GetGroupTunnelLimit(ctx context.Context, groupID int64) (int, error) {
	group, err := s.getCachedGroupByID(ctx, groupID)
	if err != nil {
		return 0, err
	}
	if group == nil {
		return 0, errors.New("group not found")
	}
	return group.TunnelLimit, nil
}

// GetUserBandwidth 获取用户带宽限制
func (s *userService) GetUserBandwidth(ctx context.Context, userID int64) (int, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if user == nil {
		return 0, errors.New("user not found")
	}
	if user.Bandwidth == nil {
		return 0, nil
	}
	return *user.Bandwidth, nil
}

// GetUserTrafficQuota 获取用户流量配额
func (s *userService) GetUserTrafficQuota(ctx context.Context, userID int64) (int64, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if user == nil {
		return 0, errors.New("user not found")
	}
	if user.TrafficQuota == nil {
		return 0, nil
	}
	return *user.TrafficQuota, nil
}

// getCachedGroupByID 内部方法，用于获取 Group 信息 (不再缓存)
func (s *userService) getCachedGroupByID(ctx context.Context, groupID int64) (*repository.Group, error) {
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return group, nil
}

// GetUserGroup 获取用户所属的用户组
func (s *userService) GetUserGroup(ctx context.Context, userID int64) (*repository.Group, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("用户不存在")
	}

	return s.getCachedGroupByID(ctx, user.GroupID)
}

// ResetToken 重置用户token
func (s *userService) ResetToken(ctx context.Context, identifier, password string) (*repository.User, error) {
	// 尝试通过用户名或邮箱获取用户
	var user *repository.User
	var err error

	// 先尝试用户名登录
	user, err = s.GetByUsername(ctx, identifier)
	if errors.Is(err, redis.Nil) || (err != nil && err.Error() == "用户不存在") {
		user, err = s.userRepo.GetByEmail(ctx, identifier)
		if err != nil {
			return nil, errors.New("用户不存在")
		}
	} else if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("用户不存在")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("密码错误")
	}

	// 生成新的32位随机token
	user.Token = rand.String(32)

	// 更新用户token
	if err := s.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// AdminResetToken 管理员重置用户token（不需要密码验证）
func (s *userService) AdminResetToken(ctx context.Context, user *repository.User) error {
	// 生成新的32位随机token
	user.Token = rand.String(32)

	// 更新用户token
	if err := s.Update(ctx, user); err != nil {
		return err
	}

	return nil
}

// GetGroupTraffic 获取用户组的流量
func (s *userService) GetGroupTraffic(ctx context.Context, groupID int64) (int64, error) {
	group, err := s.getCachedGroupByID(ctx, groupID)
	if err != nil {
		return 0, err
	}
	if group == nil {
		return 0, errors.New("group not found")
	}
	return group.TrafficQuota, nil
}

// GetUserUsedTraffic 获取用户已使用的流量
func (s *userService) GetUserUsedTraffic(ctx context.Context, userID int64) (int64, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if user == nil {
		return 0, errors.New("user not found")
	}

	// 通过用户名获取流量日志
	trafficLog, err := s.userTrafficRepo.GetByUsername(ctx, user.Username)
	if err != nil {
		s.logger.Error("Failed to get user traffic log", "error", err)
		return 0, err
	}

	// 如果没有流量记录，返回0
	if trafficLog == nil {
		return 0, nil
	}

	return trafficLog.TotalTraffic, nil
}

// Count 获取用户总数
func (s *userService) Count(ctx context.Context) (int64, error) {
	return s.userRepo.Count(ctx)
}

// IsUserBlacklisted 检查用户是否在黑名单中（GroupID为6）
func (s *userService) IsUserBlacklisted(ctx context.Context, userID int64) (bool, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, errors.New("user not found")
	}

	// 黑名单用户组ID为6
	return user.GroupID == 6, nil
}

// IsUserBlacklistedByUsername 根据用户名检查用户是否在黑名单中
func (s *userService) IsUserBlacklistedByUsername(ctx context.Context, username string) (bool, error) {
	user, err := s.GetByUsername(ctx, username)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, errors.New("user not found")
	}

	// 黑名单用户组ID为6
	return user.GroupID == 6, nil
}

// IsUserBlacklistedByToken 根据Token检查用户是否在黑名单中
func (s *userService) IsUserBlacklistedByToken(ctx context.Context, token string) (bool, error) {
	user, err := s.GetByToken(ctx, token)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, errors.New("user not found")
	}

	// 黑名单用户组ID为6
	return user.GroupID == 6, nil
}

// SearchUsers 搜索用户
func (s *userService) SearchUsers(ctx context.Context, keyword string) ([]*repository.User, error) {
	return s.userRepo.SearchUsers(ctx, keyword)
}
