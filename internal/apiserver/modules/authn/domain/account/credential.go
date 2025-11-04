package account

import (
	"time"
)

// CredentialType 凭据类型
type CredentialType string

const (
	CredPassword     CredentialType = "password"       // 用户名+密码
	CredPhoneOTP     CredentialType = "phone_otp"      // 手机号+短信码（OTP 不落库）
	CredOAuthWxMinip CredentialType = "oauth_wx_minip" // wx.login
	CredOAuthWecom   CredentialType = "oauth_wecom"    // qwx.login / 扫码
)

type Credential struct {
	ID        int64
	AccountID int64

	// —— 外部身份三元组：仅 OAuth/Phone 有值；password 留空 —— //
	IDP           *string // "wechat"|"wecom"|"phone" | nil(本地)
	IDPIdentifier string  // unionid | openid@appid | open_userid | +E164 | ""(password)
	AppID         *string // wechat=appid | wecom=corp_id | nil(本地)

	// —— 三件套（仅 password 会使用；其余类型为空） —— //
	Material   []byte  // PHC 哈希（password）；其余类型 NULL
	Algo       *string // "argon2id"/"bcrypt"；其余类型 NULL
	ParamsJSON []byte  // 低频元数据（如 wx.profile / wecom.agentid / phone 场景）

	// —— 通用状态；只有 password 实际用到失败计数/锁定 —— //
	Status         CredentialStatus
	FailedAttempts int        // 失败尝试次数
	LockedUntil    *time.Time // 锁定截止时间
	LastSuccessAt  *time.Time // 最近成功时间
	LastFailureAt  *time.Time // 最近失败时间

	Rev int64 // 乐观锁
}
