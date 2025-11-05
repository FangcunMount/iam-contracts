# 认证中心 - Token 管理

> 详细介绍 JWT Token 的生命周期、密钥轮换机制和黑名单管理

📖 [返回主文档](./README.md)

---

## Token 管理

### 1. Token 生命周期

```text
┌─────────────────────────────────────────────────────┐
│             Token Lifecycle Management               │
├─────────────────────────────────────────────────────┤
│                                                      │
│  1. 签发 (Issue)                                     │
│     - 用户登录成功                                    │
│     - 生成 Access + Refresh Token                    │
│                                                      │
│  2. 使用 (Use)                                       │
│     - 业务服务验证 Token                              │
│     - 通过 JWKS 获取公钥                              │
│     - 验证签名、过期时间                              │
│                                                      │
│  3. 刷新 (Refresh)                                   │
│     - Access Token 即将过期                          │
│     - 使用 Refresh Token 换取新 Token                │
│     - 旧 Token 加入黑名单                             │
│                                                      │
│  4. 撤销 (Revoke)                                    │
│     - 用户登出                                        │
│     - 管理员强制下线                                  │
│     - Token 加入黑名单                                │
│     - 删除 Redis 会话                                 │
│                                                      │
│  5. 过期 (Expire)                                    │
│     - Token 自然过期                                  │
│     - Redis TTL 自动清理                              │
│                                                      │
└─────────────────────────────────────────────────────┘
```

### 5.2 密钥轮换机制

```text
时间线：
─────────────────────────────────────────────────────►
                                                       
2025-09-01  K-2025-09 生成并开始签发
            │
            ▼
2025-10-01  K-2025-10 生成并开始签发 (当前)
            K-2025-09 进入 Grace Period (仅验证)
            │
            ▼
2025-10-08  K-2025-09 过期，从 JWKS 移除
            │
            ▼
2025-11-01  K-2025-11 生成并开始签发
            K-2025-10 进入 Grace Period
            │
            ▼
2025-11-08  K-2025-10 过期，从 JWKS 移除
```

**轮换策略**:

```go
// 伪代码
type KeyRotationPolicy struct {
    RotationInterval time.Duration  // 30 days
    GracePeriod      time.Duration  // 7 days
    MinKeysInJWKS    int            // 1 (current)
    MaxKeysInJWKS    int            // 2 (current + grace)
}

func (p *KeyRotationPolicy) ShouldRotate(currentKey *Key) bool {
    return time.Since(currentKey.CreatedAt) >= p.RotationInterval
}

func (p *KeyRotationPolicy) ShouldRemove(key *Key) bool {
    return time.Since(key.CreatedAt) >= p.RotationInterval + p.GracePeriod
}
```

### 5.3 黑名单管理

```go
// 添加到黑名单
func RevokeToken(ctx context.Context, jti string, exp time.Time) error {
    ttl := time.Until(exp)
    if ttl <= 0 {
        return nil // 已过期，无需加黑名单
    }
    
    key := fmt.Sprintf("blacklist:%s", jti)
    return redis.Set(ctx, key, "revoked", ttl).Err()
}

// 检查黑名单
func IsRevoked(ctx context.Context, jti string) (bool, error) {
    key := fmt.Sprintf("blacklist:%s", jti)
    val, err := redis.Get(ctx, key).Result()
    
    if err == redis.Nil {
        return false, nil // 不在黑名单
    }
    if err != nil {
        return false, err
    }
    
    return true, nil // 在黑名单
}
```
