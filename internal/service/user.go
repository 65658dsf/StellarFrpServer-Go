package service

import (
	"context"
	"errors"
	"fmt"
	"stellarfrp/internal/repository"
	"stellarfrp/pkg/async"
	"stellarfrp/pkg/email"
	"stellarfrp/pkg/logger"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/util/rand"
)

// UserService 用户服务接口
type UserService interface {
	Create(ctx context.Context, user *repository.User) error
	CreateAsync(ctx context.Context, user *repository.User) (string, error)
	GetByID(ctx context.Context, id int64) (*repository.User, error)
	GetByUsername(ctx context.Context, username string) (*repository.User, error)
	GetByEmail(ctx context.Context, email string) (*repository.User, error)
	Update(ctx context.Context, user *repository.User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, page, pageSize int) ([]*repository.User, error)
	GetTaskStatus(ctx context.Context, taskID string) (string, error)
	SendEmail(ctx context.Context, email, msgType string) error
	Login(ctx context.Context, identifier, password string) (*repository.User, error)
	GetGroupName(ctx context.Context, groupID int64) (string, error)
}

// userService 用户服务实现
type userService struct {
	userRepo    repository.UserRepository
	groupRepo   repository.GroupRepository
	redisClient *redis.Client
	emailSvc    *email.Service
	logger      *logger.Logger
	worker      *async.Worker
}

// NewUserService 创建用户服务实例
func NewUserService(
	userRepo repository.UserRepository,
	groupRepo repository.GroupRepository,
	redisClient *redis.Client,
	worker *async.Worker,
	emailSvc *email.Service,
	logger *logger.Logger,
) UserService {
	return &userService{
		userRepo:    userRepo,
		groupRepo:   groupRepo,
		redisClient: redisClient,
		worker:      worker,
		emailSvc:    emailSvc,
		logger:      logger,
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
	return s.userRepo.GetByID(ctx, id)
}

// Update 更新用户信息
func (s *userService) Update(ctx context.Context, user *repository.User) error {
	return s.userRepo.Update(ctx, user)
}

// Delete 删除用户
func (s *userService) Delete(ctx context.Context, id int64) error {
	return s.userRepo.Delete(ctx, id)
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
		user, userErr := s.userRepo.GetByEmail(ctx, email)
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
	return s.userRepo.GetByUsername(ctx, username)
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
	user, err = s.userRepo.GetByUsername(ctx, identifier)
	if err != nil && err.Error() == "user not found" {
		// 如果用户名不存在，尝试邮箱登录
		user, err = s.userRepo.GetByEmail(ctx, identifier)
		if err != nil {
			return nil, errors.New("用户不存在")
		}
	} else if err != nil {
		return nil, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("密码错误")
	}

	// 只有在用户没有token或token为空时才生成新的token
	if user.Token == "" {
		// 生成固定的32位随机token
		user.Token = rand.String(32)

		// 更新用户token
		if err := s.userRepo.Update(ctx, user); err != nil {
			return nil, err
		}
	}

	return user, nil
}

// GetGroupName 根据组ID获取组名称
func (s *userService) GetGroupName(ctx context.Context, groupID int64) (string, error) {
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		s.logger.Error("Failed to get group", "error", err)
		return "未知用户组", err
	}
	return group.Name, nil
}
