# iam / 认证中心（AuthN）模块落地蓝图（单体内集成）

> 目标：在 **`internal/apiserver` 单体**内落地“认证中心”模块，遵循现有三层结构（领域 / 应用 / 基础设施），与「基础用户」「权限中心」并列协作，不拆分独立服务。

---

## 顶层结构（在你现有结构上新增 ✓）

```text
internal/apiserver/
├─ domain/
│  └─ authn/
│     ├─ account/
│     │  ├─ account.go
│     │  ├─ credential.go
│     │  ├─ wechat.go
│     │  └─ vo.go
│     ├─ port/                      # 这里只放接口
│     │  ├─ repo.go                 # AccountRepo/WeChatRepo/OperationRepo/UoW
│     │  ├─ signer.go               # JWTSigner/JWTVerifier
│     │  ├─ refresh_store.go        # RefreshStore
│     │  └─ blacklist.go            # Blacklist（可选）
│     ├─ service/                   # 这里放“实现”
│     │  ├─ token_issuer.go         # 实现：TokenIssuer
│     │  ├─ refresh.go              # 实现：RefreshService
│     │  └─ account_domain.go       # 可选：ensureAccount/mergeByUnionID 等
│     └─ policy/
│        ├─ policy.go               # Policy 相关逻辑
│        ├─ policy_test.go          # Policy 相关测试
│        └─ provider.go             # Provider 接口（策略扩展点，仅接口）
├─ application/
│  └─ authn/
│     ├─ login.go                 # Login 用例（Provider → find/create → IssueAccess+CreateRefresh）
│     ├─ token.go                 # Refresh 用例（Rotate → IssueAccess）
│     ├─ logout.go                # Logout/Revoke（可选黑名单）
│     └─ provider/
│        ├─ basic.go              # BasicProvider（用户名/口令）
│        └─ wx_minip.go           # WeChat 小程序 Provider
├─ infra/
│  ├─ mysql/
│  │  └─ authn/
│  │     ├─ po.go                 # GORM PO：AuthAccountPO/WeChatAccountPO/PasswordPO
│  │     ├─ mapper.go             # PO<->DO 映射
│  │     ├─ repo_account.go       # AccountRepo 实现（支持 Tx）
│  │     ├─ repo_wechat.go        # WeChatRepo 实现
│  │     ├─ repo_operation.go     # OperationRepo 实现
│  │     └─ uow.go                # UnitOfWork（GORM 事务注入 Tx 版 repo）
│  ├─ redis/
│  │  └─ authn/
│  │     ├─ refresh_store.go      # RefreshStore 实现 + Lua 旋转
│  │     └─ blacklist.go          # Blacklist 实现（可选）
│  └─ jwt/
│     ├─ signer_config.go         # KeySet 配置加载（kid/alg/public_jwk/kms_key）
│     └─ signer.go                # JWTSigner（RS/ES/EdDSA）
├─ interface/
│  └─ http/
│     └─ authn/
│        ├─ handlers.go           # /auth/login /auth/token /auth/logout /jwks
│        └─ router.go             # 路由注册
├─ options/
│  └─ authn.go                    # TTL/KeySet 的配置结构体与默认值
└─ config/
   └─ authn.yaml                  # JWT/KeySet/TTL 配置样例
```

---

## 领域层（核心类型与端口）

### 1) Account 聚合根 & 子实体（`domain/authn/account/*.go`）

```go
package account

type Provider string
const (
    ProviderPassword Provider = "op:password"
    ProviderWeChat   Provider = "wx:minip"
)

type Status int8
const (StatusDisabled Status = 0; StatusActive Status = 1)

type Account struct {
    ID, UserID       string
    Provider         Provider
    ExternalID       string
    AppID, UnionID   *string
    Status           Status
    CreatedAt, UpdatedAt time.Time
}
func (a *Account) Disable() { a.Status = StatusDisabled }
func (a *Account) IsActive() bool { return a.Status == StatusActive }

// 子实体（非聚合根）
type OperationCredential struct {
    AccountID string
    Username  string
    PasswordHash []byte
    Algo      string   // argon2id/bcrypt
    Params    []byte   // JSON
    FailedAttempts int
    LockedUntil    *time.Time
    LastChangedAt  time.Time
}

type WeChatAccount struct {
    AccountID string
    AppID     string
    OpenID    string
    UnionID   *string
    Nickname  *string
    AvatarURL *string
    Meta      []byte
    CreatedAt time.Time
}

// 最小 JWT 声明
type AccessClaims struct {
    Sub string `json:"sub"`
    Aid string `json:"aid"`
    Aud string `json:"aud"`
    Iss string `json:"iss"`
    Iat int64  `json:"iat"`
    Exp int64  `json:"exp"`
    Jti string `json:"jti"`
    Kid string `json:"kid"`
    Sid string `json:"sid,omitempty"`
}
```

### 2) 端口（`domain/authn/port/*.go`）

```go
package port

type AccountRepo interface {
    FindByRef(ctx context.Context, provider account.Provider, externalID string, appID *string) (*account.Account, error)
    Create(ctx context.Context, a *account.Account) error
    Disable(ctx context.Context, id string) error
}

type WeChatRepo interface {
    Upsert(ctx context.Context, wx *account.WeChatAccount) error
    FindByAppOpenID(ctx context.Context, appID, openid string) (*account.WeChatAccount, error)
}

type OperationRepo interface {
    GetByUsername(ctx context.Context, username string) (*account.OperationCredential, error)
    Create(ctx context.Context, cred *account.OperationCredential) error
    UpdateHash(ctx context.Context, username string, hash []byte, algo string, params []byte) error
    IncFailAndMaybeLock(ctx context.Context, username string, maxFail int, lockFor time.Duration) (int, error)
    ResetFailures(ctx context.Context, username string) error
}

type TxRepos struct { Accounts AccountRepo; WeChat WeChatRepo; Operation OperationRepo }

type UnitOfWork interface { WithinTx(ctx context.Context, fn func(tx TxRepos) error) error }

// 签名/刷新/黑名单
type JWTSigner interface { SignAccess(claims map[string]any) (jwt, jti, kid string, err error); Now() time.Time }

type RefreshStore interface { Save(ctx context.Context, hash []byte, payload []byte, ttl time.Duration) error; Load(ctx context.Context, hash []byte) ([]byte, error); Delete(ctx context.Context, hash []byte) error }

type Blacklist interface { BlockJTI(ctx context.Context, jti string, ttl time.Duration) error; IsBlocked(ctx context.Context, jti string) (bool, error) }
```

### 3) 领域服务端口（`domain/authn/service/*.go`）

```go
package service

type AuthInput map[string]any

type AccountRef struct {
    Provider   account.Provider
    ExternalID string
    AppID, UnionID *string
    Profile    map[string]any
}

// 认证提供者接口
type Provider interface {
    Name() string
    Authenticate(ctx context.Context, in AuthInput) (AccountRef, error)
}

// 令牌签发接口
type TokenIssuer interface {
    IssueAccess(ctx context.Context, userID, accountID, audience, sid string, ttl time.Duration) (jwt, jti string, exp time.Time, err error)
}

// 刷新令牌服务接口
type RefreshService interface {
    Create(ctx context.Context, userID, accountID, sid string, ttl time.Duration) (refreshPlain string, err error)
    Rotate(ctx context.Context, refreshPlain string, ttl time.Duration) (userID, accountID, sid, newRefresh string, err error)
}
```

---

## 应用层（`application/authn`）

### 1) Login 用例（编排）

```go
// application/authn/login.go

type LoginCmd struct { Provider string; Input service.AuthInput; Audience, DeviceID string }

type TokenPair struct { AccessToken, TokenType string; ExpiresIn int; RefreshToken, JTI string }

type LoginService struct {
    Reg *Registry          // Provider 注册表
    UoW port.UnitOfWork
    Acc port.AccountRepo
    WX  port.WeChatRepo
    Op  port.OperationRepo
    Issuer service.TokenIssuer
    Refresh service.RefreshService
    Cfg interface{ AccessTTL(aud string) time.Duration; RefreshTTL() time.Duration }
}

func (s *LoginService) Login(ctx context.Context, in LoginCmd) (TokenPair, error) {
    prov, ok := s.Reg.Get(in.Provider); if !ok { return TokenPair{}, perrors.BadRequest("unknown_provider") }
    ref, err := prov.Authenticate(ctx, in.Input); if err != nil { return TokenPair{}, perrors.Unauthorized("invalid_credentials") }

    // 幂等 find-or-create（事务 + 子实体 upsert）
    acc, err := s.Acc.FindByRef(ctx, ref.Provider, ref.ExternalID, ref.AppID)
    if errors.Is(err, infra.ErrNotFound) {
        err = s.UoW.WithinTx(ctx, func(tx port.TxRepos) error {
            acc = &account.Account{ID:newID(),UserID:newID(),Provider:ref.Provider,ExternalID:ref.ExternalID,AppID:ref.AppID,UnionID:ref.UnionID,Status:account.StatusActive,CreatedAt:now(),UpdatedAt:now()}
            if err := tx.Accounts.Create(ctx, acc); err != nil { return err }
            if ref.Provider == account.ProviderWeChat {
                wx := &account.WeChatAccount{AccountID:acc.ID,AppID:*ref.AppID,OpenID:ref.ExternalID,UnionID:ref.UnionID,Meta:mustJSON(ref.Profile),CreatedAt:now()}
                return tx.WeChat.Upsert(ctx, wx)
            }
            return nil
        })
        if err != nil { return TokenPair{}, err }
    } else if err != nil { return TokenPair{}, err }
    if !acc.IsActive() { return TokenPair{}, perrors.Forbidden("account_disabled") }

    sid := chooseSID(in.DeviceID)
    jwt, jti, exp, err := s.Issuer.IssueAccess(ctx, acc.UserID, acc.ID, in.Audience, sid, s.Cfg.AccessTTL(in.Audience))
    if err != nil { return TokenPair{}, err }
    rt, err := s.Refresh.Create(ctx, acc.UserID, acc.ID, sid, s.Cfg.RefreshTTL()); if err != nil { return TokenPair{}, err }
    return TokenPair{AccessToken:jwt,TokenType:"Bearer",ExpiresIn:int(exp.Sub(time.Now()).Seconds()),RefreshToken:rt,JTI:jti}, nil
}
```

### 2) Refresh 用例

```go
// application/authn/token.go

type RefreshCmd struct { RefreshToken string; Audience string }

type TokenService struct { Issuer service.TokenIssuer; Refresh service.RefreshService; Cfg interface{ AccessTTL(aud string) time.Duration; RefreshTTL() time.Duration } }

func (s *TokenService) Refresh(ctx context.Context, in RefreshCmd) (TokenPair, error) {
    uid, aid, sid, newRT, err := s.Refresh.Rotate(ctx, in.RefreshToken, s.Cfg.RefreshTTL())
    if err != nil { return TokenPair{}, perrors.Unauthorized("invalid_refresh") }
    jwt, jti, exp, err := s.Issuer.IssueAccess(ctx, uid, aid, in.Audience, sid, s.Cfg.AccessTTL(in.Audience))
    if err != nil { return TokenPair{}, err }
    return TokenPair{AccessToken:jwt,TokenType:"Bearer",ExpiresIn:int(exp.Sub(time.Now()).Seconds()),RefreshToken:newRT,JTI:jti}, nil
}
```

### 3) Provider 实现（示例：Basic）

```go
// application/authn/provider/basic.go

type BasicProvider struct { Op port.OperationRepo }
func (p *BasicProvider) Name() string { return "basic" }
func (p *BasicProvider) Authenticate(ctx context.Context, in service.AuthInput) (service.AccountRef, error) {
    u, _ := in["username"].(string); pw, _ := in["password"].(string)
    cred, err := p.Op.GetByUsername(ctx, u); if err != nil { return service.AccountRef{}, perrors.Unauthorized("invalid_credentials") }
    if isLocked(cred) { return service.AccountRef{}, perrors.Forbidden("account_locked") }
    if !verifyPassword(pw, cred) { _, _ = p.Op.IncFailAndMaybeLock(ctx, u, 5, 10*time.Minute); return service.AccountRef{}, perrors.Unauthorized("invalid_credentials") }
    _ = p.Op.ResetFailures(ctx, u)
    return service.AccountRef{Provider: account.ProviderPassword, ExternalID: u}, nil
}
```

---

## 基础设施层（MySQL/Redis/JWT）

### 1) MySQL PO/Repo（GORM）

```go
// infra/mysql/authn/po.go

type AuthAccountPO struct {
    ID string `gorm:"primaryKey;type:char(36)"`
    UserID string `gorm:"type:char(36);not null"`
    Provider string `gorm:"size:32;not null"`
    ExternalID string `gorm:"size:128;not null"`
    AppID *string `gorm:"size:64"`
    UnionID *string `gorm:"size:128"`
    Status int8 `gorm:"not null;default:1"`
    CreatedAt time.Time
    UpdatedAt time.Time
}
func (AuthAccountPO) TableName() string { return "auth_accounts" }

// 复合唯一索引（在迁移或 AutoMigrate 钩子中创建）
// CREATE UNIQUE INDEX uq_provider_app_external ON auth_accounts(provider, app_id, external_id);
```

```go
// infra/mysql/authn/repo_account.go

type AccountRepoMySQL struct { DB *gorm.DB }
func (r *AccountRepoMySQL) FindByRef(ctx context.Context, provider account.Provider, externalID string, appID *string) (*account.Account, error) {
    var po AuthAccountPO
    q := r.DB.WithContext(ctx).Where("provider=? AND external_id=?", provider, externalID)
    if appID == nil { q = q.Where("app_id IS NULL") } else { q = q.Where("app_id=?", *appID) }
    if err := q.First(&po).Error; err != nil { if errors.Is(err, gorm.ErrRecordNotFound) { return nil, infra.ErrNotFound }; return nil, err }
    return mapToDO(po), nil
}
func (r *AccountRepoMySQL) Create(ctx context.Context, a *account.Account) error { return r.DB.WithContext(ctx).Create(mapToPO(a)).Error }
func (r *AccountRepoMySQL) Disable(ctx context.Context, id string) error { return r.DB.WithContext(ctx).Model(&AuthAccountPO{}).Where("id=?", id).Update("status", 0).Error }
```

> `repo_wechat.go` / `repo_operation.go` 类似实现，`uow.go` 使用 `db.Transaction` 注入 Tx 绑定的 repo。

### 2) Redis：RefreshStore 与 Lua 旋转

```go
// infra/redis/authn/refresh_store.go

func (s *RefreshStoreRedis) Rotate(ctx context.Context, oldPlain, newPlain string, payload []byte, ttl time.Duration) error {
    oldKey := sha256Hex(oldPlain); newKey := sha256Hex(newPlain)
    // EVAL Lua: GET old → DEL old → SETEX new payload ttl
}
```

### 3) JWT：KeySet 配置与签发

- `infra/jwt/signer_config.go`：加载 `authn.jwt.keyset`（只含公钥与 KMS key 引用，不存私钥）。
- `infra/jwt/signer.go`：用 RS/ES/EdDSA 签名，JWT header 写 `kid`；提供 `/.well-known/jwks.json` 的 handler（从配置导出）。

---

## 接口层（`interface/http/authn`）

```go
// handlers.go（节选）
func (h *Handler) PostLogin(c *gin.Context)  { /* bind → app.Login → JSON */ }
func (h *Handler) PostToken(c *gin.Context)  { /* bind → app.Token.Refresh */ }
func (h *Handler) PostLogout(c *gin.Context) { /* optional: blacklist / revoke */ }
func (h *Handler) GetJWKS(c *gin.Context)    { /* serve jwks from config */ }

// router.go
func Register(r *gin.Engine, h *Handler) {
    g := r.Group("/auth")
    g.POST("/login", h.PostLogin)
    g.POST("/token", h.PostToken)
    g.POST("/logout", h.PostLogout)
    r.GET("/.well-known/jwks.json", h.GetJWKS)
}
```

---

## 配置（`options/authn.go` + `config/authn.yaml`）

```go
type AuthNOptions struct {
  AccessTTL struct{ Web, Admin, Mobile time.Duration }
  RefreshTTL time.Duration
  JWT struct {
    Alg string
    JWKSCacheTTL time.Duration
    OverlapWindow time.Duration
    KeySet []struct { Kid string; Active bool; KMSKey string; PublicJWK map[string]any; NotBefore, NotAfter time.Time }
  }
}
```

---

## 接口—实现映射（速查）

| 接口 | 归属 | 实现 |
|---|---|---|
| `AccountRepo` | domain/authn/port | `infra/mysql/authn/repo_account.go` |
| `WeChatRepo` | domain/authn/port | `infra/mysql/authn/repo_wechat.go` |
| `OperationRepo` | domain/authn/port | `infra/mysql/authn/repo_operation.go` |
| `UnitOfWork` | domain/authn/port | `infra/mysql/authn/uow.go` |
| `Provider` | domain/authn/service | `application/authn/provider/*` |
| `TokenIssuer` | domain/authn/service | `application/authn/*` 调 `port.JWTSigner` |
| `JWTSigner` | domain/authn/port | `infra/jwt/signer.go` |
| `RefreshService` | domain/authn/service | `application/authn/token.go`（调 `port.RefreshStore`） |
| `RefreshStore` | domain/authn/port | `infra/redis/authn/refresh_store.go` |

---

## 数据库迁移（migrations/000X_authn.sql）

```sql
CREATE TABLE auth_accounts (
  id           CHAR(36) PRIMARY KEY,
  user_id      CHAR(36) NOT NULL,
  provider     VARCHAR(32)  NOT NULL,
  external_id  VARCHAR(128) NOT NULL,
  app_id       VARCHAR(64)  NULL,
  union_id     VARCHAR(128) NULL,
  status       TINYINT NOT NULL DEFAULT 1,
  created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_provider_app_external (provider, app_id, external_id)
);

CREATE TABLE auth_wechat_accounts (
  account_id  CHAR(36) PRIMARY KEY,
  app_id      VARCHAR(64)  NOT NULL,
  openid      VARCHAR(128) NOT NULL,
  union_id    VARCHAR(128) NULL,
  nickname    VARCHAR(64)  NULL,
  avatar_url  VARCHAR(255) NULL,
  meta        JSON         NULL,
  created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_wx_acc FOREIGN KEY (account_id) REFERENCES auth_accounts(id) ON DELETE CASCADE,
  UNIQUE KEY uq_wx (app_id, openid)
);

CREATE TABLE auth_passwords (
  username        VARCHAR(128)   PRIMARY KEY,
  account_id      CHAR(36)       NOT NULL UNIQUE,
  password_hash   VARBINARY(255) NOT NULL,
  algo            VARCHAR(16)    NOT NULL,
  params          JSON           NULL,
  failed_attempts INT            NOT NULL DEFAULT 0,
  locked_until    TIMESTAMP      NULL,
  last_changed_at TIMESTAMP      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  must_reset      TINYINT        NOT NULL DEFAULT 0,
  created_at      TIMESTAMP      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_pwd_acc FOREIGN KEY (account_id) REFERENCES auth_accounts(id) ON DELETE CASCADE
);
```

---

## 接入步骤（建议）

1. **落库**：执行 migrations，AutoMigrate/手动迁移均可。
2. **注入**：在 `container/` 里装配 repo、UoW、JWTSigner、RefreshStore、Provider Registry、Login/Token 服务、HTTP handler。
3. **路由**：在 `routers.go` 注册 `interface/http/authn` 的路由组。
4. **切流**：资源路由只接受 Bearer，中间件从 Basic 中剥离；Basic 仅用于 `/auth/login` Provider。
5. **配置**：放置 `config/authn.yaml`，在 `options/authn.go` 加载并校验。

---

### 说明

- 本蓝图保持与你现有 User/Child/Guardianship 的组织方式一致：**实体/VO/端口在领域层**、**编排在应用层**、**持久化在 infra**。有利于跨模块协作与统一风格。
- JWT KeySet 使用**配置驱动**；可逐步升级到 DB 审计与自动轮换，不影响外部契约。
- WeCom QR、黑名单、审计/指标等可在此基础上增量接入。
