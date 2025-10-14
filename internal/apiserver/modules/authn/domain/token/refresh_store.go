package port

import (
	"context"
	"time"
)

// RefreshStore —— 刷新票据存储抽象（建议用 Redis 实现）
// 键：sha256(refresh_plain) → payload(JSON){user_id, account_id, sid, created_at}，TTL=RefreshTTL
// 强烈建议提供原子“旋转”以满足一次刷新即废旧票据的语义。
type RefreshStore interface {
	// Save：保存新票据（key=sha256(plain)）
	Save(ctx context.Context, key []byte, payload []byte, ttl time.Duration) error

	// Load：按 key 读取 payload；不存在返回 (nil, nil)
	Load(ctx context.Context, key []byte) ([]byte, error)

	// Delete：按 key 删除
	Delete(ctx context.Context, key []byte) error

	// Rotate：原子“删旧设新”（DEL old → SETEX new）—— 建议用 Lua 实现
	Rotate(ctx context.Context, oldKey, newKey, newPayload []byte, ttl time.Duration) error

	// DeleteAllByUser：可选，用于“全端下线”（返回删除数量）
	DeleteAllByUser(ctx context.Context, userID string) (int, error)
}
