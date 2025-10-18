# 认证中心 - 安全设计

> 详细介绍密码安全、防重放攻击、速率限制等安全机制

📖 [返回主文档](./README.md)

---

## 安全设计

### 1. 密码安全

```go
// 密码哈希
func HashPassword(password string) (string, error) {
    cost := 12 // BCrypt cost factor
    hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
    return string(hash), err
}

// 密码验证
func VerifyPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

**密码策略**:

- ✅ 最小长度: 8 字符
- ✅ 复杂度要求: 大小写字母 + 数字 + 特殊字符
- ✅ 密码历史: 不能重复使用最近 5 次密码
- ✅ 过期策略: 90 天强制修改
- ✅ 失败锁定: 5 次失败后锁定 15 分钟

### 6.2 防重放攻击

```go
// OAuth 2.0 PKCE (Proof Key for Code Exchange)
type PKCEChallenge struct {
    CodeVerifier  string // 随机字符串（43-128字符）
    CodeChallenge string // SHA256(CodeVerifier) 的 Base64URL
    Method        string // "S256"
}

// 授权请求
func AuthorizeWithPKCE(challenge PKCEChallenge) (authCode string) {
    // 存储 challenge 到 Redis (TTL: 10min)
    redis.Set(authCode, challenge.CodeChallenge, 10*time.Minute)
    return authCode
}

// Token 请求（必须提供 verifier）
func ExchangeTokenWithPKCE(authCode, verifier string) (*Token, error) {
    storedChallenge := redis.Get(authCode)
    computedChallenge := base64url.Encode(sha256.Sum256(verifier))
    
    if storedChallenge != computedChallenge {
        return nil, errors.New("PKCE verification failed")
    }
    
    // 签发 Token
    return issueToken(userID)
}
```

### 6.3 速率限制

```go
// 基于 Token Bucket 算法
type RateLimiter struct {
    Capacity int           // 桶容量
    Rate     time.Duration // 补充速率
}

// 登录速率限制
// - 同一 IP: 10次/分钟
// - 同一账号: 5次/分钟

// 伪代码
func CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) error {
    count := redis.Incr(ctx, key)
    
    if count == 1 {
        redis.Expire(ctx, key, window)
    }
    
    if count > limit {
        return errors.ErrTooManyRequests
    }
    
    return nil
}
```

### 6.4 HTTPS Only

```text
┌─────────────────────────────────────────────────────┐
│              TLS/HTTPS Configuration                 │
├─────────────────────────────────────────────────────┤
│                                                      │
│  - TLS 1.2+ 强制                                     │
│  - HSTS (Strict-Transport-Security) 启用             │
│  - Certificate Pinning 客户端可选                    │
│  - 证书自动续期 (Let's Encrypt)                      │
│                                                      │
│  Nginx 配置示例:                                     │
│  ssl_protocols TLSv1.2 TLSv1.3;                     │
│  ssl_ciphers HIGH:!aNULL:!MD5;                      │
│  add_header Strict-Transport-Security               │
│    "max-age=31536000; includeSubDomains" always;    │
│                                                      │
└─────────────────────────────────────────────────────┘
```

---

## 7. API 设计

### 7.1 认证 API

```http
# 微信小程序登录
POST /api/v1/auth/wechat:login
Content-Type: application/json

{
  "code": "051Ab2ll2QMRCH05o2nl2vhOX64Ab2lx",
  "device_id": "iPhone13_iOS16"
}

