package wechatapp

import (
	"context"
	"time"
)

// ================== External Service Interfaces (Driven Ports) ==================
// 定义领域模型所依赖的外部服务接口，由基础设施层提供实现

// AccessTokenCache 微信应用访问令牌缓存接口
type AccessTokenCache interface {
	// 访问令牌缓存操作
	Get(ctx context.Context, appID string) (*AppAccessToken, error)
	// 设置访问令牌缓存
	Set(ctx context.Context, appID string, aat *AppAccessToken, ttl time.Duration) error
	// 尝试获取单飞刷新锁
	TryLockRefresh(ctx context.Context, appID string, ttl time.Duration) (ok bool, unlock func(), err error)
}

// SecretVault 秘钥保险库接口
// 基础设施适配层需提供实现（本地 AES-GCM 或云 KMS）
type SecretVault interface {
	// Encrypt 加密
	Encrypt(ctx context.Context, plaintext []byte) (cipher []byte, err error)
	// Decrypt 解密
	Decrypt(ctx context.Context, cipher []byte) (plaintext []byte, err error)
	// Sign 签名（托管签名 KMS/HSM）
	Sign(ctx context.Context, keyRef string, data []byte) (sig []byte, err error)
}

// AppTokenProvider 微信应用访问令牌提供器接口
type AppTokenProvider interface {
	// Fetch 获取访问令牌
	Fetch(ctx context.Context, app *WechatApp) (*AppAccessToken, error)
}
