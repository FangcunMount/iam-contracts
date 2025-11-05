package authentication

// 统一原始入参（不传领域实体）
type AuthInput struct {
	TenantID  *int64
	RemoteIP  string
	UserAgent string

	// password
	Username string
	Password string

	// phone_otp
	PhoneE164 string
	OTP       string

	// wx_minip
	WxAppID  string
	WxJsCode string

	// wecom
	WecomCorpID string
	WecomCode   string
	WecomState  string

	// jwt_token
	AccessToken string // JWT 访问令牌
}
