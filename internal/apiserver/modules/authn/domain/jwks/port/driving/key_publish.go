package driving

import (
	"context"
	"time"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/jwks"
)

// CacheTag 缓存标签（用于 HTTP 缓存）
type CacheTag struct {
	// ETag 实体标签（内容哈希）
	ETag string

	// LastModified 最后修改时间
	LastModified time.Time
}

// KeySetPublishService JWKS 发布服务接口
// 负责构建和发布 /.well-known/jwks.json
// 由应用层调用，实现在领域服务层
type KeySetPublishService interface {
	// BuildJWKS 构建 JWKS JSON
	// 查询所有可发布的密钥（Active + Grace 状态且未过期）
	// 返回：JWKS JSON 字节流和缓存标签
	BuildJWKS(ctx context.Context) (jwksJSON []byte, tag CacheTag, err error)

	// GetPublishableKeys 获取可发布的密钥列表
	// 用于预览或调试
	GetPublishableKeys(ctx context.Context) ([]*jwks.Key, error)

	// ValidateCacheTag 验证缓存标签
	// 用于 HTTP 304 Not Modified 响应
	// clientTag: 客户端提供的 ETag 或 Last-Modified
	// 返回：true 表示缓存有效（未变更）
	ValidateCacheTag(ctx context.Context, clientTag CacheTag) (bool, error)

	// GetCurrentCacheTag 获取当前缓存标签
	// 用于生成 HTTP 响应头
	GetCurrentCacheTag(ctx context.Context) (CacheTag, error)

	// RefreshCache 刷新缓存
	// 用于强制更新缓存（密钥轮换后）
	RefreshCache(ctx context.Context) error
}

// JWKSResponse JWKS HTTP 响应
type JWKSResponse struct {
	// JWKS JWKS 对象
	JWKS jwks.JWKS

	// CacheTag 缓存标签
	CacheTag CacheTag

	// MaxAge 缓存最大有效期（秒）
	MaxAge int
}
