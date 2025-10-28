package wechatsession

import (
	"context"
	"errors"
	"time"
)

// WechatSession 微信会话信息
type WechatSession struct {
	AppID     string    // 微信应用 AppID
	OpenID    string    // 普通用户的标识，对当前公众号/小程序唯一
	UnionID   *string   // 用户在开放平台的唯一标识符
	Ver       int       // 版本号
	SKCipher  []byte    // session_key 密文（不落明文）
	UpdatedAt time.Time // 最后更新时间
}

// NewWechatSession 创建微信会话信息领域对象
func NewWechatSession(appid, openid string, opts ...WechatSessionOption) *WechatSession {
	session := &WechatSession{
		AppID:  appid,
		OpenID: openid,
	}
	for _, opt := range opts {
		opt(session)
	}
	return session
}

// WechatSessionOption 微信会话信息选项
type WechatSessionOption func(*WechatSession)

func WithWechatSessionUnionID(unionid string) WechatSessionOption {
	return func(s *WechatSession) { s.UnionID = &unionid }
}
func WithWechatSessionVersion(ver int) WechatSessionOption {
	return func(s *WechatSession) { s.Ver = ver }
}
func WithWechatSessionSKCipher(cipher []byte) WechatSessionOption {
	return func(s *WechatSession) { s.SKCipher = cipher }
}
func WithWechatSessionUpdatedAt(t time.Time) WechatSessionOption {
	return func(s *WechatSession) { s.UpdatedAt = t }
}

// ===== SecretVault（端口）：统一加/解密 & 托管签名 =====
// 基础设施适配层需提供实现（本地 AES-GCM 或云 KMS）
type SecretVault interface {
	Encrypt(ctx context.Context, plaintext []byte) (cipher []byte, err error)
	Decrypt(ctx context.Context, cipher []byte) (plaintext []byte, err error)
	Sign(ctx context.Context, keyRef string, data []byte) (sig []byte, err error) // 托管签名（KMS/HSM）
}

// 轮换 session_key（wx.login 后）
func (s *WechatSession) Rotate(vault SecretVault, newPlain []byte, now time.Time) error {
	// 验证明文合法性
	if len(newPlain) == 0 {
		return errors.New("empty session_key")
	}
	c, err := vault.Encrypt(context.Background(), newPlain)
	if err != nil {
		return err
	}
	s.SKCipher = c
	s.Ver++
	s.UpdatedAt = now
	return nil
}

// 基于软 TTL 评估是否“可能陈旧”
func (s *WechatSession) IsSoftExpired(now time.Time, softTTL time.Duration) bool {
	return now.After(s.UpdatedAt.Add(softTTL))
}
