# AuthN SDK 日志使用说明

## 概述

AuthN SDK 现已集成结构化日志功能，使用项目统一的日志库 `github.com/FangcunMount/component-base/pkg/log`。

## 日志级别

SDK 使用以下日志级别：

- **Info**: 重要操作（连接、初始化、验证成功）
- **Debug**: 详细的调试信息（Token 解析、Key 查找、缓存操作）
- **Warn**: 警告信息（验证失败、Key 未找到）
- **Error**: 错误信息（连接失败、HTTP 请求失败、解析错误）

## 日志内容

### Client 初始化和连接

```go
client, err := authnsdk.NewClient(ctx, "localhost:8081")
```

**日志输出：**

```text
INFO  [AuthN SDK] Connecting to IAM authn gRPC endpoint: localhost:8081
DEBUG [AuthN SDK] Using insecure credentials for gRPC connection
INFO  [AuthN SDK] Successfully connected to IAM authn gRPC endpoint
```

### JWKS 管理器

#### 初始化

```go
verifier, err := authnsdk.NewVerifier(cfg, client)
```

**日志输出：**

```text
INFO  [AuthN SDK] Initializing verifier with JWKS URL: https://iam.example.com/.well-known/jwks.json
INFO  [AuthN SDK] Remote verification enabled
INFO  [AuthN SDK] Initializing JWKS manager with URL: https://iam.example.com/.well-known/jwks.json, refresh interval: 5m0s
```

#### JWKS 刷新

**首次加载：**

```text
DEBUG [AuthN SDK] Refreshing JWKS from https://iam.example.com/.well-known/jwks.json
INFO  [AuthN SDK] Successfully refreshed JWKS, loaded 2 keys
```

**使用 ETag 缓存：**

```text
DEBUG [AuthN SDK] Refreshing JWKS from https://iam.example.com/.well-known/jwks.json
DEBUG [AuthN SDK] Using ETag: "abc123"
DEBUG [AuthN SDK] JWKS not modified (304), using cached keys
```

#### Key 查找

```text
DEBUG [AuthN SDK] Looking up key with kid: key-2024-01
DEBUG [AuthN SDK] Found key for kid: key-2024-01
```

**Key 未找到时重试：**

```text
DEBUG [AuthN SDK] Looking up key with kid: new-key
DEBUG [AuthN SDK] Key not found for kid new-key, refreshing JWKS and retrying
DEBUG [AuthN SDK] Refreshing JWKS from https://iam.example.com/.well-known/jwks.json
INFO  [AuthN SDK] Successfully refreshed JWKS, loaded 3 keys
DEBUG [AuthN SDK] Found key for kid new-key after refresh
```

### Token 验证

#### 成功的本地验证

```go
resp, err := verifier.Verify(ctx, token, nil)
```

**日志输出：**

```text
DEBUG [AuthN SDK] Starting token verification
DEBUG [AuthN SDK] Looking up key with kid: key-2024-01
DEBUG [AuthN SDK] Found key for kid: key-2024-01
DEBUG [AuthN SDK] Parsing JWT token
DEBUG [AuthN SDK] Local verification successful, subject: user123, user_id: 456
INFO  [AuthN SDK] Token verification completed successfully
```

#### 带远程验证

```go
resp, err := verifier.Verify(ctx, token, &authnsdk.VerifyOptions{ForceRemote: true})
```

**日志输出：**

```text
DEBUG [AuthN SDK] Starting token verification
DEBUG [AuthN SDK] Parsing JWT token
DEBUG [AuthN SDK] Local verification successful, subject: user123, user_id: 456
DEBUG [AuthN SDK] Calling remote verification
DEBUG [AuthN SDK] Remote verification successful
INFO  [AuthN SDK] Token verification completed successfully
```

#### 验证失败

```go
resp, err := verifier.Verify(ctx, invalidToken, nil)
```

**日志输出：**

```text
DEBUG [AuthN SDK] Starting token verification
DEBUG [AuthN SDK] Parsing JWT token
DEBUG [AuthN SDK] JWT parse failed: token is expired
WARN  [AuthN SDK] Local token verification failed: token is expired
```

### 连接关闭

```go
client.Close()
```

**日志输出：**

```text
DEBUG [AuthN SDK] Closing gRPC connection
DEBUG [AuthN SDK] gRPC connection closed successfully
```

## 日志配置

SDK 使用项目的全局日志配置。可以通过配置日志级别来控制输出详细程度：

### 开发环境

```yaml
log:
  level: debug  # 输出所有日志，包括详细的调试信息
```

### 生产环境

```yaml
log:
  level: info   # 只输出重要信息，减少日志量
```

## 日志前缀

所有 SDK 日志都使用 `[AuthN SDK]` 前缀，便于在应用日志中识别和过滤：

```bash
# 过滤 AuthN SDK 日志
grep "\[AuthN SDK\]" application.log

# 只看错误日志
grep "\[AuthN SDK\].*ERROR" application.log
```

## 性能考虑

- Debug 级别的日志在生产环境会被自动过滤，不影响性能
- 关键操作（连接、验证成功）使用 Info 级别
- 敏感信息（完整 Token）不会被记录
- 只记录 Token 的 kid、subject、user_id 等元数据

## 示例：完整的验证流程日志

```text
INFO  [AuthN SDK] Connecting to IAM authn gRPC endpoint: localhost:8081
DEBUG [AuthN SDK] Using insecure credentials for gRPC connection
INFO  [AuthN SDK] Successfully connected to IAM authn gRPC endpoint
INFO  [AuthN SDK] Initializing verifier with JWKS URL: https://iam.example.com/.well-known/jwks.json
INFO  [AuthN SDK] Remote verification enabled
INFO  [AuthN SDK] Initializing JWKS manager with URL: https://iam.example.com/.well-known/jwks.json, refresh interval: 5m0s
DEBUG [AuthN SDK] Starting token verification
DEBUG [AuthN SDK] No keys in cache, refreshing JWKS
DEBUG [AuthN SDK] Refreshing JWKS from https://iam.example.com/.well-known/jwks.json
INFO  [AuthN SDK] Successfully refreshed JWKS, loaded 2 keys
DEBUG [AuthN SDK] Looking up key with kid: key-2024-01
DEBUG [AuthN SDK] Found key for kid: key-2024-01
DEBUG [AuthN SDK] Parsing JWT token
DEBUG [AuthN SDK] Local verification successful, subject: user123, user_id: 456
INFO  [AuthN SDK] Token verification completed successfully
```

## 故障排查

### 连接问题

查找包含 "Connecting" 或 "Failed to connect" 的日志：

```bash
grep "Connecting\|Failed to connect" application.log
```

### JWKS 问题

查找 JWKS 相关错误：

```bash
grep "\[AuthN SDK\].*JWKS" application.log | grep -E "ERROR|WARN"
```

### 验证失败了

查找验证失败的原因：

```bash
grep "\[AuthN SDK\].*verification failed" application.log
```
