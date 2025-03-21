package types

// SendMessageRequest 发送验证码请求
type SendMessageRequest struct {
	Email string `json:"email" binding:"required"`
	Type  string `json:"type" binding:"required,oneof=register reset_password"`
	// 极验验证参数
	LotNumber     string `json:"lot_number"`
	CaptchaOutput string `json:"captcha_output"`
	PassToken     string `json:"pass_token"`
	GenTime       string `json:"gen_time"`
	// 验证对象，用于兼容前端传递的验证参数
	Validate *ValidateParams `json:"validate"`
}

// ValidateParams 验证参数对象
type ValidateParams struct {
	CaptchaID     string `json:"captcha_id"`
	LotNumber     string `json:"lot_number"`
	PassToken     string `json:"pass_token"`
	GenTime       string `json:"gen_time"`
	CaptchaOutput string `json:"captcha_output"`
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Email    string `json:"email" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Password string `json:"password" binding:"required"`
}
