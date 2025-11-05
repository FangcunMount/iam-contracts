package authentication

// 选择哪种认证策略
type Scenario string

const (
	AuthPassword Scenario = "password"
	AuthPhoneOTP Scenario = "phone_otp"
	AuthWxMinip  Scenario = "oauth_wx_minip"
	AuthWecom    Scenario = "oauth_wecom"
)

// AMR（认证方法引用），用于审计与 Step-Up
type AMR string

const (
	AMRPassword AMR = "pwd"
	AMROTP      AMR = "otp"
	AMRWx       AMR = "wechat"
	AMRWecom    AMR = "wecom"
)

// 认证主体（输出，用于签 Token/授权）
type Principal struct {
	AccountID int64
	UserID    int64
	TenantID  *int64
	AMR       []string
	Claims    map[string]any
}
