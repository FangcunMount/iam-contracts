package wechatapp

import "time"

// AppAccessToken 微信应用访问令牌
type AppAccessToken struct {
	Token     string
	ExpiresAt time.Time
}

// 提前 skew 进行续期判断（比如 120s）
func (t *AppAccessToken) IsValid(now time.Time, skew time.Duration) bool {
	return t != nil && t.Token != "" && now.Add(skew).Before(t.ExpiresAt)
}
