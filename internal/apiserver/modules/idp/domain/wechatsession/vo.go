package wechatsession

// ExternalClaim 外部身份声明（OAuth2 / OIDC 标准字段 + 扩展字段）
type ExternalClaim struct {
	Provider     string // wechat_miniprogram / apple / google ...
	AppID        string
	Subject      string // openid/ sub
	UnionID      *string
	DisplayName  *string
	AvatarURL    *string
	Phone        *string
	Email        *string
	ExpiresInSec int
}
