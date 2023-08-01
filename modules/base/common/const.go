package common

// CodeType 验证码类型
type CodeType int

const (
	// CodeTypeRegister 注册
	CodeTypeRegister CodeType = iota
	// CodeTypePayPWD 支付密码
	CodeTypePayPWD
	// CodeTypeForgetLoginPWD 忘记登录密码
	CodeTypeForgetLoginPWD
	// CodeTypeCheckMobile 校验指定手机号是否正确
	CodeTypeCheckMobile
	// DestroyAccount 注销账号
	CodeTypeDestroyAccount
)

const (
	// CacheKeySMSCode 短信验证码的缓存key
	CacheKeySMSCode string = "smscode:"
)
