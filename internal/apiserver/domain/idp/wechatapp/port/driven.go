package port

import (
	"context"
	"time"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/idp/wechatapp"
)

// ==================== Driven Ports (驱动端口) ====================
// 定义领域模型、领域服务所依赖的外部服务接口
// 基础设施适配层需提供实现

// WechatAppRepository 微信应用存储库接口
type WechatAppRepository interface {
	// 创建接口
	Create(ctx context.Context, app *domain.WechatApp) error

	// 查询接口
	GetByID(ctx context.Context, id idutil.ID) (*domain.WechatApp, error)
	GetByAppID(ctx context.Context, appID string) (*domain.WechatApp, error)

	// 更新接口
	Update(ctx context.Context, app *domain.WechatApp) error
}

// AccessTokenCache 微信应用访问令牌缓存接口
type AccessTokenCache interface {
	// 访问令牌缓存操作
	Get(ctx context.Context, appID string) (*domain.AppAccessToken, error)
	// 设置访问令牌缓存
	Set(ctx context.Context, appID string, aat *domain.AppAccessToken, ttl time.Duration) error
	// 尝试获取单飞刷新锁
	TryLockRefresh(ctx context.Context, appID string, ttl time.Duration) (ok bool, unlock func(), err error)
}

// ===== SecretVault（端口）：统一加/解密 & 托管签名 =====

// 基础设施适配层需提供实现（本地 AES-GCM 或云 KMS）
type SecretVault interface {
	// Encrypt 加密
	Encrypt(ctx context.Context, plaintext []byte) (cipher []byte, err error)
	// Decrypt 解密
	Decrypt(ctx context.Context, cipher []byte) (plaintext []byte, err error)
	// Sign 签名
	Sign(ctx context.Context, keyRef string, data []byte) (sig []byte, err error) // 托管签名（KMS/HSM）
}

// ===== AppTokenProvider（端口）：微信应用访问令牌提供器 =====
// AppTokenProvider 微信应用访问令牌提供器接口
type AppTokenProvider interface {
	// Fetch 获取访问令牌
	Fetch(ctx context.Context, app *domain.WechatApp) (*domain.AppAccessToken, error)
}
