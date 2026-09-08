package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

// LoginStatePurgeResult 只输出固定状态族与汇总计数，不包含键名或令牌。
type LoginStatePurgeResult struct {
	Families map[string]RefreshTokenPurgeResult `json:"families"`
}

// PurgeLoginState 只清理 IAM 登录状态。调用前必须停止签发和会话写入。
// Redis 可能被其他服务共享，禁止使用 FLUSHDB 或接收用户自定义扫描模式。
func PurgeLoginState(ctx context.Context, client goredis.UniversalClient, batchSize int64, apply bool) (LoginStatePurgeResult, error) {
	result := LoginStatePurgeResult{Families: map[string]RefreshTokenPurgeResult{}}
	if client == nil || batchSize <= 0 {
		return result, fmt.Errorf("Redis 客户端和正数批次大小必填")
	}
	for _, family := range []struct{ name, pattern string }{
		{"session", sessionKeyspace.Prefix("*")},
		{"user_session_index", userSessionIndexKeyspace.Prefix("*")},
		{"login_identity_session_index", loginIdentitySessionIndexKeyspace.Prefix("*")},
		{"refresh_token", refreshTokenKeyspace.Prefix("*")},
		{"consumed_refresh_token", consumedRefreshTokenKeyspace.Prefix("*")},
		{"revoked_access_token", revokedAccessTokenKeyspace.Prefix("*")},
	} {
		var counts RefreshTokenPurgeResult
		var cursor uint64
		for {
			keys, next, err := client.Scan(ctx, cursor, family.pattern, batchSize).Result()
			if err != nil {
				return result, fmt.Errorf("登录状态扫描失败: %s", family.name)
			}
			counts.Scanned += int64(len(keys))
			counts.Matched += int64(len(keys))
			if apply && len(keys) > 0 {
				deleted, err := client.Unlink(ctx, keys...).Result()
				if err != nil {
					return result, fmt.Errorf("登录状态清理失败: %s", family.name)
				}
				counts.Deleted += deleted
			}
			result.Families[family.name] = counts
			cursor = next
			if cursor == 0 {
				break
			}
		}
	}
	return result, nil
}
