package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"path/filepath"
	"time"

	"stellarfrp/pkg/logger"
)

// Config 邮件配置
type Config struct {
	Host     string // SMTP服务器地址
	Port     int    // SMTP服务器端口
	Username string // 邮箱账号
	Password string // 邮箱密码
	From     string // 发件人
	FromName string // 发件人名称
}

// EmailType 邮件类型
type EmailType string

const (
	// TypeRegister 注册验证码邮件
	TypeRegister EmailType = "register"
	// TypeResetPassword 重置密码邮件
	TypeResetPassword EmailType = "reset_password"
	// TypeWelcome 欢迎邮件
	TypeWelcome EmailType = "register_success"
)

// EmailData 邮件数据
type EmailData struct {
	To          string    // 收件人
	Subject     string    // 邮件主题
	VerifyCode  string    // 验证码
	ExpireTime  time.Time // 过期时间
	ProductName string    // 产品名称
	UserName    string    // 用户名
}

// Service 邮件服务
type Service struct {
	config Config
	logger *logger.Logger
}

// NewService 创建邮件服务
func NewService(config Config, logger *logger.Logger) *Service {
	return &Service{
		config: config,
		logger: logger,
	}
}

// SendEmail 发送邮件
func (s *Service) SendEmail(emailType EmailType, data EmailData) error {
	// 设置默认产品名称
	if data.ProductName == "" {
		data.ProductName = "StellarFrp-恒星映射"
	}

	// 根据邮件类型设置主题
	if data.Subject == "" {
		switch emailType {
		case TypeRegister:
			data.Subject = fmt.Sprintf("%s - 注册验证码", data.ProductName)
		case TypeResetPassword:
			data.Subject = fmt.Sprintf("%s - 重置密码验证码", data.ProductName)
		}
	}

	// 渲染邮件内容
	content, err := s.renderTemplate(string(emailType), data)
	if err != nil {
		return fmt.Errorf("渲染邮件模板失败: %w", err)
	}

	// 发送邮件
	return s.send(data.To, data.Subject, content)
}

// renderTemplate 渲染邮件模板
func (s *Service) renderTemplate(templateName string, data EmailData) (string, error) {
	// 获取模板路径
	templateFile := fmt.Sprintf("%s.html", templateName)
	tmplPath := filepath.Join("templates", "email", templateFile)

	// 解析模板
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return "", fmt.Errorf("解析邮件模板失败: %w", err)
	}

	// 渲染模板
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		return "", fmt.Errorf("执行邮件模板失败: %w", err)
	}

	return buf.String(), nil
}

// send 发送邮件
func (s *Service) send(to, subject, body string) error {
	// 设置邮件头
	header := make(map[string]string)
	header["From"] = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.From)
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=UTF-8"

	// 组装邮件内容
	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// 设置认证信息
	auth := smtp.PlainAuth(
		"",
		s.config.Username,
		s.config.Password,
		s.config.Host,
	)

	// 连接到SMTP服务器
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// 配置TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         s.config.Host,
	}

	// 创建TLS连接
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("创建TLS连接失败: %w", err)
	}

	// 创建SMTP客户端
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("创建SMTP客户端失败: %w", err)
	}
	defer client.Close()

	// 身份认证
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP认证失败: %w", err)
	}

	// 设置发件人和收件人
	if err = client.Mail(s.config.From); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("设置收件人失败: %w", err)
	}

	// 发送邮件内容
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("准备发送数据失败: %w", err)
	}
	_, err = w.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("关闭数据写入失败: %w", err)
	}

	if err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	s.logger.Info(fmt.Sprintf("邮件已发送至 %s", to))
	return nil
}

// SendVerificationCode 发送验证码邮件
func (s *Service) SendVerificationCode(to, code string, expireMinutes int) error {
	expireTime := time.Now().Add(time.Duration(expireMinutes) * time.Minute)

	data := EmailData{
		To:         to,
		VerifyCode: code,
		ExpireTime: expireTime,
	}

	return s.SendEmail(TypeRegister, data)
}

// SendPasswordResetCode 发送密码重置验证码邮件
func (s *Service) SendPasswordResetCode(to, userName, code string, expireMinutes int) error {
	expireTime := time.Now().Add(time.Duration(expireMinutes) * time.Minute)

	data := EmailData{
		To:         to,
		UserName:   userName,
		VerifyCode: code,
		ExpireTime: expireTime,
	}

	return s.SendEmail(TypeResetPassword, data)
}

// SendWelcomeEmail 发送欢迎邮件
func (s *Service) SendWelcomeEmail(to, userName string) error {
	data := EmailData{
		To:          to,
		UserName:    userName,
		ProductName: "StellarFrp-恒星映射",
		Subject:     "欢迎加入StellarFrp",
	}

	return s.SendEmail(TypeWelcome, data)
}
