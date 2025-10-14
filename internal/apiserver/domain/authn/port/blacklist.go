package port

import (
	"context"
	"time"
)

// Blacklist（可选）—— 立刻让当前 Access 失效（按 jti 封禁）
// 如果不需要“即时下线”，可以不实现此接口，保持完全无状态。
type Blacklist interface {
	// BlockJTI：封禁指定 jti，TTL=剩余 access 寿命
	BlockJTI(ctx context.Context, jti string, ttl time.Duration) error

	// IsBlocked：查询 jti 是否被封禁（资源侧或 /auth/verify 可调用）
	IsBlocked(ctx context.Context, jti string) (bool, error)
}
