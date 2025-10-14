# 认证中心（AuthN Only）— 单一聚合根 + 多子实体 设计文档

> 本文为：采用“**单一聚合根 + 多子实体**”的领域建模；通过 **Provider 策略模式**扩展 Basic / WeChat MiniProgram / WeCom QR；**Access=JWT（配置驱动 KeySet）**、**Refresh=Redis（旋转）**、黑名单可选。

---

## 1) 边界与默认（Scope & Defaults）

- **只做 AuthN**：完成“外部凭证 → 系统用户”的认证闭环与访问令牌签发；**不承载权限（AuthZ）**。
- **Access Token = JWT**：短期（默认 30m）；无状态、仅最小身份（不含权限）。
- **Refresh Token = 随机串**：长期（默认 30d）；仅存 Redis，**旋转**（一次刷新即废旧票据）。
- **黑名单（可选）**：按 `jti` 立刻失效当前 Access；若不需要“立刻下线”，则保持完全无状态。
- **密钥治理（配置驱动）**：KeySet 放配置（kid/alg/public_jwk/KMS alias/not_before/not_after/active），零停机轮换；未来可无感升级到 DB 元数据。
- **速率限制**：`/auth/login`、`/auth/token`：`5r/10s`（按 ip 或 ip+account/provider）。

---

## 2) 领域模型（单一聚合根 + 子实体）

### 2.1 聚合根：Account（统一锚点）

```go
type Provider string
const (
    ProviderPassword Provider = "op:password"
    ProviderWeChat   Provider = "wx:minip"
    ProviderWeCom    Provider = "wecom:qr"
)

type Status int8
const (StatusDisabled Status = 0; StatusActive Status = 1)

type Account struct {
    ID, UserID     string
    Provider       Provider
    ExternalID     string   // username / openid / open_userid
    AppID, UnionID *string  // wechat/wecom use
    Status         Status
    CreatedAt, UpdatedAt time.Time
}
func (a *Account) Disable() { a.Status = StatusDisabled }
func (a *Account) IsActive() bool { return a.Status == StatusActive }
```

#### 不变式

- 唯一性：`UNIQUE(provider, app_id, external_id)`（对无 app_id 的 Provider 退化为 `(provider, external_id)`）。
- 业务规则：`Status=Disabled ⇒ 禁止登录/刷新`。

### 2.2 子实体（受控，非聚合根）

```go
// 运营口令（OperationAccount）
type OperationAccount struct {
    AccountID     string
    Username      string
    PasswordHash  []byte     // Argon2id（推荐）或 bcrypt
    Algo          string
    Params        []byte     // JSON
    FailedAttempts int
    LockedUntil   *time.Time
    LastChangedAt time.Time
}

// 微信（WeChatAccount）
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

// 企业微信（WeComAccount）
type WeComAccount struct {
    AccountID  string
    CorpID     string
    UserID     string
    OpenUserID *string
    Name       *string
    AvatarURL  *string
    DeptIDs    []int
    Meta       []byte
    CreatedAt  time.Time
}
```

> 子实体仅在**认证阶段**参与校验与画像维护；对下游 Token/刷新/审计保持统一的 `Account` 视角（`sub/aid`）。

---

## 3) 端口（Ports / Anti-Corruption）

```go
// 统一锚点
type AccountRepo interface {
    FindByRef(ctx context.Context, provider Provider, externalID string, appID *string) (*Account, error)
    Create(ctx context.Context, a *Account) error
    Disable(ctx context.Context, id string) error
}

// 子实体各自 repo（受控，非聚合根）
type OperationRepo interface {
    GetByUsername(ctx context.Context, username string) (*OperationCredential, error)
    Create(ctx context.Context, cred *OperationCredential) error
    UpdateHash(ctx context.Context, username string, hash []byte, algo string, params []byte) error
    IncFailAndMaybeLock(ctx context.Context, username string, maxFail int, lockFor time.Duration) (int, error)
    ResetFailures(ctx context.Context, username string) error
}

type WeChatRepo interface {
    Upsert(ctx context.Context, wx *WeChatAccount) error
    FindByAppOpenID(ctx context.Context, appID, openid string) (*WeChatAccount, error)
}

type WeComRepo interface {
    Upsert(ctx context.Context, wc *WeComAccount) error
    FindByCorpUser(ctx context.Context, corpID, userID string) (*WeComAccount, error)
}

// Token 签发与刷新（与存储解耦）
type TokenIssuer interface {
    IssueAccess(ctx context.Context, userID, accountID, audience, sid string, ttl time.Duration) (jwt string, jti string, exp time.Time, err error)
}

type RefreshService interface {
    Create(ctx context.Context, userID, accountID, sid string, ttl time.Duration) (refreshPlain string, err error)
    Rotate(ctx context.Context, refreshPlain string, ttl time.Duration) (userID, accountID, sid, newRefresh string, err error)
    Revoke(ctx context.Context, refreshPlain string) error
    RevokeAllByUser(ctx context.Context, userID string) (int, error)
}

type Blacklist interface {
    BlockJTI(ctx context.Context, jti string, ttl time.Duration) error
    IsBlocked(ctx context.Context, jti string) (bool, error)
}

type RateLimiter interface { Allow(ctx context.Context, key string, n int) bool }
```

---

## 4) Unit of Work（事务边界）

统一的事务封装，保证“**一次登录首登** → `Account` + 子实体**原子**落库”。

```go
// 事务内可见的 repo 集合
type TxRepos struct {
    Accounts  AccountRepo
    WeChat    WeChatRepo
    WeCom     WeComRepo
    Operation OperationRepo
}

type UnitOfWork interface {
    WithinTx(ctx context.Context, fn func(tx TxRepos) error) error
}
```

> MySQL 实现建议：每个 Repo 同时持有 `*sql.DB` 和可选的 `*sql.Tx`；内部优先使用 Tx。

---

## 5) Provider 策略层（插件化）

### 5.1 接口与注册表

```go
type AuthInput map[string]any

type Provider interface {
    Name() string  // "basic" | "wx:minip" | "wecom:qr"
    Authenticate(ctx context.Context, in AuthInput) (AccountRef, error)
}

type AccountRef struct {
    Provider   Provider
    ExternalID string
    AppID      *string
    UnionID    *string
    Profile    map[string]any // 画像/扩展（供子实体 upsert）
}

type ProviderRegistry interface { Get(name string) (Provider, bool) }
```

### 5.2 统一登录入口（应用服务）

```go
type ProviderLoginCmd struct {
    Provider string
    Input    AuthInput
    Audience string
    DeviceID string
}

type TokenPair struct {
    AccessToken  string `json:"access_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int    `json:"expires_in"`
    RefreshToken string `json:"refresh_token,omitempty"`
    JTI          string `json:"jti"`
}

func (s *AuthApp) Login(ctx context.Context, in ProviderLoginCmd) (TokenPair, error) {
    prov, ok := s.reg.Get(in.Provider)
    if !ok { return TokenPair{}, fmt.Errorf("unknown provider: %s", in.Provider) }

    // 1) 外部认证
    ref, err := prov.Authenticate(ctx, in.Input)
    if err != nil { return TokenPair{}, ErrInvalidCredentials }

    // 2) 幂等查找或创建（捕获唯一键冲突）
    acc, err := s.acc.FindByRef(ctx, ref.Provider, ref.ExternalID, ref.AppID)
    if errors.Is(err, ErrNotFound) {
        err = s.uow.WithinTx(ctx, func(tx TxRepos) error {
            acc = &Account{
                ID: newID(), UserID: s.resolveUser(ctx, ref), Provider: ref.Provider,
                ExternalID: ref.ExternalID, AppID: ref.AppID, UnionID: ref.UnionID,
                Status: StatusActive, CreatedAt: now(), UpdatedAt: now(),
            }
            if err := tx.Accounts.Create(ctx, acc); err != nil {
                if isDup(err) {
                    var e error
                    acc, e = tx.Accounts.FindByRef(ctx, ref.Provider, ref.ExternalID, ref.AppID)
                    return e
                }
                return err
            }
            switch ref.Provider { // 回写子实体
            case ProviderWeChat:
                wx := &WeChatAccount{AccountID: acc.ID, AppID: deref(ref.AppID), OpenID: ref.ExternalID, UnionID: ref.UnionID,
                    Nickname: optStr(ref.Profile["nickname"]), AvatarURL: optStr(ref.Profile["avatar"]),
                    Meta: mustJSON(ref.Profile), CreatedAt: now(),
                }
                return tx.WeChat.Upsert(ctx, wx)
            case ProviderWeCom:
                wc := &WeComAccount{AccountID: acc.ID, CorpID: deref(ref.AppID), UserID: ref.ExternalID,
                    OpenUserID: optStrPtr(ref.Profile["open_userid"]), Name: optStrPtr(ref.Profile["name"]),
                    AvatarURL: optStrPtr(ref.Profile["avatar"]), DeptIDs: optIntSlice(ref.Profile["dept_ids"]),
                    Meta: mustJSON(ref.Profile), CreatedAt: now(),
                }
                return tx.WeCom.Upsert(ctx, wc)
            }
            return nil
        })
        if err != nil { return TokenPair{}, err }
    } else if err != nil { return TokenPair{}, err }

    if !acc.IsActive() { return TokenPair{}, ErrAccountDisabled }

    // 3) 签发 & 刷新
    sid := chooseSID(in.DeviceID)
    jwt, jti, exp, err := s.issuer.IssueAccess(ctx, acc.UserID, acc.ID, in.Audience, sid, s.cfg.AccessTTL(in.Audience))
    if err != nil { return TokenPair{}, err }
    rt, err := s.refresh.Create(ctx, acc.UserID, acc.ID, sid, s.cfg.RefreshTTL())
    if err != nil { return TokenPair{}, err }

    return TokenPair{AccessToken: jwt, TokenType: "Bearer", ExpiresIn: int(exp.Sub(now()).Seconds()), RefreshToken: rt, JTI: jti}, nil
}
```

> `basic` Provider 内部只做用户名/口令校验（OperationRepo），对外仍返回 `AccountRef{Provider:"op:password", ExternalID: username}`，后续流程一致。

---

## 6) 存储模型（DDL 摘要）

```sql
-- 统一锚点
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

-- 微信子实体（1:1）
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

-- 企业微信子实体（1:1）
CREATE TABLE auth_wecom_accounts (
  account_id  CHAR(36) PRIMARY KEY,
  corp_id     VARCHAR(64)  NOT NULL,
  userid      VARCHAR(64)  NOT NULL,
  open_userid VARCHAR(128) NULL,
  name        VARCHAR(64)  NULL,
  avatar_url  VARCHAR(255) NULL,
  dept_ids    JSON         NULL,
  meta        JSON         NULL,
  created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_wecom_acc FOREIGN KEY (account_id) REFERENCES auth_accounts(id) ON DELETE CASCADE,
  UNIQUE KEY uq_wecom (corp_id, COALESCE(open_userid, userid))
);

-- 运营口令（1:1）
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

## 7) Redis 键与 Lua 旋转（Refresh & Blacklist）

- `RT:<sha256(refresh_plain)> → {user_id, account_id, sid, created_at}`（TTL=30d）
- `BL:<jti> → 1`（TTL=剩余 access 寿命，可选）

### Lua 原子旋转

```lua
-- KEYS[1]=oldKey, KEYS[2]=newKey; ARGV[1]=payloadJSON, ARGV[2]=ttlSec
local v = redis.call('GET', KEYS[1])
if not v then return {err='ERR_REFRESH_NOT_FOUND'} end
redis.call('DEL', KEYS[1])
redis.call('SET', KEYS[2], ARGV[1], 'EX', ARGV[2])
return 'OK'
```

---

## 8) JWT & KeySet（配置驱动）

```yaml
authn:
  access_ttl:
    web: 30m
    admin: 10m
    mobile: 30m
  refresh_ttl: 30d
  jwt:
    alg: RS256
    jwks_cache_ttl: 5m
    overlap_window: 60m
    keyset:
      - kid: "2025-10-13-rs256-02"
        active: true
        kms_key: "alias/jwt-2025-10-13"   # 仅引用，不放私钥明文
        public_jwk: { kty: RSA, n: "...", e: "AQAB" }
        not_before: "2025-10-13T00:00:00Z"
        not_after:  "2025-12-01T00:00:00Z"
      - kid: "2025-08-01-rs256-01"
        active: false
        kms_key: "alias/jwt-2025-08-01"
        public_jwk: { kty: RSA, n: "...", e: "AQAB" }
        not_before: "2025-08-01T00:00:00Z"
        not_after:  "2025-11-01T00:00:00Z"
```

**轮换要点**：先发布新公钥进 JWKS → 等缓存传播 → 切 `active`；并行期 ≥ `access_ttl`；多实例一致热更新。

---

## 9) 应用层契约（HTTP）

- `POST /auth/login`（统一入口）
  - Req：`{ provider, input, audience, device_id? }`
  - Resp：`TokenPair{ access_token, token_type:"Bearer", expires_in, refresh_token, jti }`
- `POST /auth/token`（刷新）
  - Req：`{ grant_type:"refresh_token", refresh_token }`
  - Resp：同上（旋转返回新 refresh）
- `POST /auth/logout`（可选）
  - Req：`{ refresh_token? , jti? }`（带 `jti` 时写黑名单）
- `POST /auth/verify`（可选，Bearer 内省）
  - Req：`{ token_or_authz }`
  - Resp：`{ active, sub, aid, aud, iat, exp, jti, kid, sid? }`
- `GET /.well-known/jwks.json` → JWKS 公钥集合

---

## 10) 错误模型与映射

```go
var (
  ErrNotFound           = errors.New("not found")
  ErrInvalidCredentials = errors.New("invalid credentials")
  ErrAccountLocked      = errors.New("account temporarily locked")
  ErrAccountDisabled    = errors.New("account disabled")
)
```

- **HTTP**：401/403/429；统一返回 `error`, `error_description`（避免区分“用户不存在/密码错误”）。

---

## 11) 安全 / 风控 / 观测

- **JWT 最小化**：`sub/aid/aud/iat/exp/jti/kid(/sid)`。
- **Pepper + Salt**：口令哈希使用服务端 pepper（KMS 注入）+ per-user salt；成功后透明重哈希。
- **限流**：`login:<provider>:<ip>`、`token_endpoint:<ip+account>`；`state/code` 做 `SeenOnce`。
- **NTP**：时钟对齐 ±30s；`iat/exp` 校验容忍度。
- **日志脱敏**：禁止打印口令/Refresh 明文；审计最小化。
- **指标**：`login_succeeded/failed`、`token_issued/refreshed`、`jwt_kid_distribution`、`jwks_cache_hit`、`rotate_events`。

---

## 12) 时序要点

- **Basic**：`Provider(basic) → OperationRepo.Verify → Account find-or-create(已有) → IssueAccess → CreateRefresh → Audit`。
- **WeChat Mini**：`jscode2session → Account find-or-create → IssueAccess → CreateRefresh → Audit`。
- **WeCom QR**：`code→identity → Account find-or-create(app_id=corp_id) → IssueAccess → CreateRefresh → Audit`。
- **刷新**：`Rotate(refresh) → IssueAccess → Audit`。

---

## 13) 实施清单（Checklist）

- [ ] Provider：`basic` / `wx:minip` / `wecom:qr` 实现与注册
- [ ] Repos：`AccountRepo` / `WeChatRepo` / `WeComRepo` / `OperationRepo` / `AuditRepo`
- [ ] UoW：`WithinTx` + Tx 版 Repo 装配
- [ ] Services：`TokenIssuer`（JWTSigner+KeySet 配置）/ `RefreshService`（Redis 旋转）
- [ ] HTTP：`/auth/login` `/auth/token` `/auth/logout` `/auth/verify` Handler
- [ ] 配置：KeySet（kid/alg/kms_key/public_jwk）+ overlap + jwks cache
- [ ] e2e：登录→刷新→旧 RT 失效→（可选）黑名单踢当前 Access

---

### 一句话

**单一聚合根 `Account`** 统一对外身份与不变式；**子实体**承载各 Provider 私有字段；配合 **Provider 策略层 + 配置驱动 JWT + Redis 刷新**，实现简洁、可演进、低耦合的认证中心。
