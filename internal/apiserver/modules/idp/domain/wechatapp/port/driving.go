package port

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp"
)

// ==================== Driving Ports (驱动端口) ====================
// 这些接口由领域层（领域服务）实现，供应用层调用
// 按照功能职责拆分，遵循接口隔离原则

// WechatAppCreator 微信应用创建器
type WechatAppCreator interface {
	Create(ctx context.Context, appID string, name string, appType domain.AppType) (*domain.WechatApp, error)
}

// CredentialManager 应用凭据管理器（分舱管理与轮换）
type CredentialManager interface {
	// 变更 AuthSecret
	ChangeAuthSecret(ctx context.Context, app *domain.WechatApp, newSecret string) error
	// 变更消息加解密密钥
	ChangeMsgSecret(ctx context.Context, app *domain.WechatApp, newSecret string) error
	// 变更 API 密钥（对称加密）
	ChangeAPISymKey(ctx context.Context, app *domain.WechatApp, newKey string) error
	// 变更 API 密钥（非对称加密）
	ChangeAPIAsymKeyWithKMS(ctx context.Context, app *domain.WechatApp, newKey string) error
	// 变更 API 密钥（明文）
	ChangeAPIAsymKeyWithPlain(ctx context.Context, app *domain.WechatApp, newKey string) error
}

// 4) 访问令牌缓存器（单飞刷新 + 过期缓冲）
type AccessTokenCacher interface {
	EnsureToken(ctx context.Context, app *domain.WechatApp) (string, error) // 单飞刷新 + 过期缓冲
}
