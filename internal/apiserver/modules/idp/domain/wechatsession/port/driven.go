package port

import (
	"context"
	"time"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession"
)

// ==================== Driven Ports (驱动端口) ====================
// 定义领域模型、领域服务所依赖的外部服务接口
// 基础设施适配层需提供实现

// WechatSessionRepository 微信会话存储库接口
type WechatSessionRepository interface {
	Get(ctx context.Context, appID, openID string) (*domain.WechatSession, error)
	Set(ctx context.Context, s *domain.WechatSession, ttl time.Duration) error
}

// ==== 基础设施：微信平台认证服务接口 ====

type AuthProvider interface {
	Code2Session(ctx context.Context, appID, appSecret, jsCode string) (Code2SessionResult, error)
	DecryptPhone(ctx context.Context, appID, appSecret, sessionKey, encryptedData, iv string) (DecryptPhoneResult, error)
}

// Code2SessionResult code2Session 返回结果
type Code2SessionResult struct {
	OpenID     string
	UnionID    string
	SessionKey string
}

// DecryptPhoneResult 解密手机号返回结果
type DecryptPhoneResult struct {
	PhoneNumber     string
	PurePhoneNumber string
	CountryCode     string
}
