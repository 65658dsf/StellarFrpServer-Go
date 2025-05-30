package constants

// 通用错误消息
const (
	// 认证相关错误
	ErrUnauthorized           = "未授权，请先登录"
	ErrInvalidToken           = "无效的Token"
	ErrInsufficientPermission = "权限不足"
	ErrAccountDisabled        = "账号已被禁用"
	ErrBlacklisted            = "您的账号已被列入黑名单，禁止访问"

	// 用户相关错误
	ErrUserNotFound      = "用户不存在"
	ErrAuthFailed        = "用户不存在或认证失败"
	ErrUsernameEmpty     = "用户名不能为空"
	ErrPasswordIncorrect = "密码错误"
	ErrUsernameExists    = "用户名已存在"
	ErrEmailExists       = "该邮箱已被注册"

	// 参数相关错误
	ErrInvalidParams  = "参数错误"
	ErrInvalidFormat  = "格式错误"
	ErrInvalidRequest = "无效请求格式"

	// 隧道相关错误
	ErrProxyNotFound     = "隧道不存在"
	ErrProxyTypeMismatch = "隧道类型不匹配"
	ErrNoNodeAccess      = "您无权使用此节点"
	ErrProxyNameFormat   = "隧道名称格式错误，应为：用户名.隧道名"
	ErrProxyNameEmpty    = "隧道名称不能为空"

	// 系统错误
	ErrInternalServer       = "服务器内部错误"
	ErrOperationTooFrequent = "请求过于频繁，请稍后重试"
	ErrTrafficExhausted     = "用户流量已耗尽"
)

// 成功消息
const (
	SuccessLogin    = "登录成功"
	SuccessRegister = "注册成功"
	SuccessCreate   = "创建成功"
	SuccessUpdate   = "更新成功"
	SuccessDelete   = "删除成功"
	SuccessGet      = "获取成功"
)
