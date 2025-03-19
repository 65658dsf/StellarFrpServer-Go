package types

// SendMessageRequest 发送验证码请求
type SendMessageRequest struct {
	Email string `json:"email" binding:"required"`
	Type  string `json:"type" binding:"required,oneof=register reset_password"`
}
