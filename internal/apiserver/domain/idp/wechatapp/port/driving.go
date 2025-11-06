package port

import (
	"context"
	"time"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/idp/wechatapp"
)

// ==================== Driving Ports (驱动端口) ====================
// 这些接口由领域层（领域服务）实现，供应用层调用
// 按照功能职责拆分，遵循接口隔离原则

// WechatAppCreator 微信应用创建器
type WechatAppCreator interface {
	Create(ctx context.Context, appID string, name string, appType domain.AppType) (*domain.WechatApp, error)
}

// WechatAppFinder 微信应用查询器
type WechatAppQuerier interface {
	QueryByAppID(ctx context.Context, appID string) (*domain.WechatApp, error)
	ExistsByAppID(ctx context.Context, appID string) (bool, error)
}

// CredentialRotater 凭据轮换器
type CredentialRotater interface {
	// 轮换凭据接口
	RotateAuthSecret(ctx context.Context, app *domain.WechatApp, newPlain string) error
	// 轮换消息加解密密钥
	RotateMsgAESKey(ctx context.Context, app *domain.WechatApp, callbackToken, encodingAESKey43 string) error
	// 轮换 API 密钥（对称加密）
	RotateAPISymKey(ctx context.Context, app *domain.WechatApp, alg domain.CryptoAlg, base64Key string) error
	// 轮换 API 密钥（非对称加密）
	RotateAPIAsymKey(ctx context.Context, app *domain.WechatApp, alg domain.CryptoAlg, kmsRef string, pubPEM []byte) error
}

// 4) 访问令牌缓存器（单飞刷新 + 过期缓冲）
type AccessTokenCacher interface {
	// 确保获取有效的访问令牌
	EnsureToken(ctx context.Context, app *domain.WechatApp, skew time.Duration) (string, error) // 单飞刷新 + 过期缓冲
}
