# Token 生命周期：颁发、刷新、撤销、JWKS

## 🎯 30 秒搞懂

### 生命周期总图

用户登录 → AccessToken + RefreshToken → 验证 / 刷新 / 撤销。JWKS 用于本地验签；在线校验额外检查撤销、Session 和准入。

## SDK 视角图

```text
                    ┌──────────────────────────┐
                    │       client.Auth()      │
                    │  SDK 里的 token 生命周期入口 │
                    └───────────┬──────────────┘
                                │
      ┌──────────────┬──────────┼──────────┬──────────────┬──────────────┐
      ↓              ↓          ↓          ↓              ↓
 VerifyToken   RefreshToken  RevokeToken  RevokeRefresh  GetJWKS
    校验            刷新          撤销         撤销刷新        取公钥
                                │
                                ↓
                      SDK 与默认服务端都已支持
```

### 工程流程图

```text
1️⃣ 用户登录发牌
   REST /authn/login
        ↓
   TokenPair{access_token, refresh_token}
        ↓
2️⃣ SDK 持有并消费
   Verify / Refresh / Revoke / GetJWKS
        ↓
3️⃣ 业务侧继续使用
   本地验签、远程校验、注销、刷新
```

### 一句话结论

这篇文档讲的是 **SDK 如何消费 token 生命周期能力**。  

### 当前能力矩阵

| 能力 | SDK 状态 | 当前服务端状态 | 说明 |
| ---- | ---- | ---- | ---- |
| 用户登录发牌 | ❌ 不在 `Auth()` 内 | ✅ 走 REST 登录链 | 见主仓库 authn 文档 |
| Access Token 校验 | ✅ 已支持 | ✅ 已实现，未装配时返回 `Unimplemented` | `VerifyToken` |
| Refresh Token 刷新 | ✅ 已支持 | ✅ 已实现，未装配时返回 `Unimplemented` | `RefreshToken` |
| Access Token 撤销 | ✅ 已支持 | ✅ 已实现，未装配时返回 `Unimplemented` | `RevokeToken` |
| Refresh Token 撤销 | ✅ 已支持 | ✅ 已实现，未装配时返回 `Unimplemented` | `RevokeRefreshToken` |
| 获取 JWKS | ✅ 已支持 | ✅ 已实现，未装配时返回 `Unimplemented` | `GetJWKS` |

### 3 行代码开始

```go
resp, err := client.Auth().RefreshToken(ctx, &authnv3.RefreshTokenRequest{
    RefreshToken: refreshToken,
})
```

---

## 1. 什么时候看这篇

适合看这篇的场景：

- 你已经拿到了 `access_token` / `refresh_token`
- 你需要在业务服务里做校验、刷新、撤销或获取 JWKS
- 你想确认服务 Token 签发的 SDK / 服务端边界

不适合只看这篇的场景：

- 用户名密码 / 微信 / OTP 登录流程本身
- REST 登录发牌链的业务设计
- JWT 本地验证的缓存、策略、降级细节
- 服务间认证 helper 的自动刷新与熔断实现

继续下钻时，优先看：

- [JWT 本地验证](./04-jwt-verification.md)
- [服务间认证](./05-service-auth.md)
- [../../../docs/02-业务模块/02-AuthN/README.md](../../../docs/02-业务模块/02-AuthN/README.md)
- [Session、Token 与 JWKS](../../../docs/02-业务模块/02-AuthN/03-Session-Token与JWKS.md)

## 2. 示例约定

除非特别说明，下面的片段默认：

- 已存在 `ctx`
- 已创建 `client`
- 已按需导入 `sdk`、`authnv3`、`errors`
- 你已经拿到了已有 token，或者明确知道自己要传的 `subject / audience / ttl`

这篇文档保留的是**最小可理解片段**。  
完整程序可以组合参考这些现有示例：

- [../_examples/basic/main.go](../_examples/basic/main.go)
- [../_examples/verifier/main.go](../_examples/verifier/main.go)
- [服务间 mTLS 与 ACL 接入](05-service-auth.md)

## 3. 两条“发牌”边界

### 3.1 用户态 TokenPair：当前不由 SDK 登录接口签发

```text
用户提交登录凭据
    ↓
REST /authn/login
    ↓
服务端 Authenticate → IssueToken
    ↓
返回 TokenPair(access_token + refresh_token)
    ↓
SDK 从这里开始消费 Verify / Refresh / Revoke / GetJWKS
```

这意味着：

- `pkg/sdk` 当前没有“用户名密码登录”或“微信登录”封装
- SDK 更像“拿到 TokenPair 之后的消费面”
- 如果你要理解登录发牌本身，应回到主仓库 authn 文档

### 3.2 服务间调用

服务间调用通过 mTLS + ACL 建立可信边界，不参与用户 Token 生命周期。

## 4. 已落地的生命周期能力

### 4.1 VerifyToken：远程校验 Access Token

```go
resp, err := client.Auth().VerifyToken(ctx, &authnv3.VerifyTokenRequest{
    AccessToken: accessToken,
})
if err != nil {
    return err
}

if !resp.Valid {
    return fmt.Errorf("token invalid: %s", resp.FailureReason)
}
```

适合场景：

- 需要直接问 IAM “这个 token 现在还有效吗”
- 你不想自己做本地验签
- 你需要服务端对 **撤销标记 / session(sid) / user-account 当前状态** 做权威判断

### 4.2 RefreshToken：用 Refresh Token 换新 TokenPair

```go
resp, err := client.Auth().RefreshToken(ctx, &authnv3.RefreshTokenRequest{
    RefreshToken: refreshToken,
})
if err != nil {
    return err
}

newAccess := resp.TokenPair.AccessToken
newRefresh := resp.TokenPair.RefreshToken
```

刷新成功后，业务侧应立刻替换旧 TokenPair，不要继续混用旧 refresh token。

### 4.3 RevokeToken / RevokeRefreshToken：主动失效

```go
_, err := client.Auth().RevokeToken(ctx, &authnv3.RevokeTokenRequest{
    AccessToken: accessToken,
})
```

```go
_, err := client.Auth().RevokeRefreshToken(ctx, &authnv3.RevokeRefreshTokenRequest{
    RefreshToken: refreshToken,
})
```

常见用法：

- 用户主动登出
- 服务端发现凭据泄漏
- 强制失效旧 token

### 4.4 GetJWKS：获取公钥集

```go
resp, err := client.Auth().GetJWKS(ctx, &authnv3.GetJWKSRequest{})
if err != nil {
    return err
}

jwksJSON := resp.Jwks
```

`GetJWKS` 更适合作为：

- 本地验签组件的远程取钥入口
- JWKS 缓存刷新的一部分

如果你是为了做本地 JWT 校验，优先看 [JWT 本地验证](./04-jwt-verification.md)。

## 5. 常见调用模式

### 5.1 “先远程校验，再继续业务”

```go
resp, err := client.Auth().VerifyToken(ctx, &authnv3.VerifyTokenRequest{
    AccessToken: accessToken,
})
if err != nil {
    return err
}
if !resp.Valid {
    return status.Error(codes.Unauthenticated, "invalid token")
}
```

### 5.2 “刷新成功后立刻替换整对 Token”

```go
resp, err := client.Auth().RefreshToken(ctx, &authnv3.RefreshTokenRequest{
    RefreshToken: refreshToken,
})
if err != nil {
    return err
}

saveTokenPair(resp.TokenPair.AccessToken, resp.TokenPair.RefreshToken)
```

### 5.3 “登出时同时撤销 access 和 refresh”

```go
_, err = client.Auth().RevokeToken(ctx, &authnv3.RevokeTokenRequest{
    AccessToken: accessToken,
})
if err != nil {
    return err
}

_, err = client.Auth().RevokeRefreshToken(ctx, &authnv3.RevokeRefreshTokenRequest{
    RefreshToken: refreshToken,
})
```

## 6. 错误处理与边界

### 6.1 当前常见错误

```go
resp, err := client.Auth().RefreshToken(ctx, &authnv3.RefreshTokenRequest{
    RefreshToken: refreshToken,
})
if err != nil {
    switch {
    case errors.IsInvalidArgument(err):
        // 请求参数不完整
    case errors.IsUnauthorized(err):
        // refresh token 无效、过期或不可用
    case errors.IsServiceUnavailable(err):
        // IAM 服务不可用
    default:
        // 其它错误
    }
    return err
}
_ = resp
```

### 6.2 当前不要讲过头的几件事

- `pkg/sdk` 当前不负责“用户登录拿 TokenPair”
- `Auth()` 已覆盖 token 生命周期消费面，但不是完整登录 SDK
- `Auth().VerifyToken(...)` 是**在线权威校验**；它看到的不只是签名和过期，还包括 `revoked_access_token`、`session(sid)`、`user/account` 当前状态
- 本地 JWKS 验签只能保证签名与时间相关声明；它**不能保证** session revoke、用户封禁、账号禁用的即时生效
- `GetJWKS` 是取钥接口，不等于完整本地验签方案；本地验签应看 [JWT 本地验证](./04-jwt-verification.md)

## 7. 继续往下读

- [快速开始](./01-quick-start.md)
- [JWT 本地验证](./04-jwt-verification.md)
- [服务间认证](./05-service-auth.md)
- [授权判定（PDP）](./06-authz.md)
- [../README.md](../README.md)
