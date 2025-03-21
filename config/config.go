package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config 应用程序配置
type Config struct {
	APIPort  int
	LogLevel string
	Database DatabaseConfig
	Redis    RedisConfig
	Email    EmailConfig
	Geetest  GeetestConfig
}

// DatabaseConfig MySQL数据库配置
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string
	Port     int
	Password string
}

// EmailConfig 邮件配置
type EmailConfig struct {
	Host     string // SMTP服务器地址
	Port     int    // SMTP服务器端口
	Username string // 邮箱账号
	Password string // 邮箱密码
	From     string // 发件人
	FromName string // 发件人名称
}

// GeetestConfig 极验验证配置
type GeetestConfig struct {
	CaptchaID  string // 验证ID
	CaptchaKey string // 验证密钥
	APIServer  string // API服务器地址
}

// Load 从环境变量加载配置
func Load() (*Config, error) {
	// 加载.env文件
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	// 解析数据库配置
	dbPort, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		dbPort = 3306 // 默认端口
	}

	// 解析Redis配置
	redisPort, err := strconv.Atoi(os.Getenv("REDIS_PORT"))
	if err != nil {
		redisPort = 6379 // 默认端口
	}

	// 解析API端口
	apiPort, err := strconv.Atoi(os.Getenv("API_PORT"))
	if err != nil {
		apiPort = 8080 // 默认端口
	}

	// 解析邮件端口
	emailPort, err := strconv.Atoi(os.Getenv("EMAIL_PORT"))
	if err != nil {
		emailPort = 587 // 默认端口
	}

	return &Config{
		APIPort:  apiPort,
		LogLevel: os.Getenv("LOG_LEVEL"),
		Database: DatabaseConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     dbPort,
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			DBName:   os.Getenv("DB_NAME"),
		},
		Redis: RedisConfig{
			Host:     os.Getenv("REDIS_HOST"),
			Port:     redisPort,
			Password: os.Getenv("REDIS_PASSWORD"),
		},
		Email: EmailConfig{
			Host:     os.Getenv("EMAIL_HOST"),
			Port:     emailPort,
			Username: os.Getenv("EMAIL_USERNAME"),
			Password: os.Getenv("EMAIL_PASSWORD"),
			From:     os.Getenv("EMAIL_FROM"),
			FromName: os.Getenv("EMAIL_FROM_NAME"),
		},
		Geetest: GeetestConfig{
			CaptchaID:  os.Getenv("GEETEST_CAPTCHA_ID"),
			CaptchaKey: os.Getenv("GEETEST_CAPTCHA_KEY"),
			APIServer:  os.Getenv("GEETEST_API_SERVER"),
		},
	}, nil
}
