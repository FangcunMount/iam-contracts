# JWT 签名方案迁移指南

> 从 HMAC 对称密钥迁移到 JWKS 非对称密钥方案

---

## 📋 变更概述

### 变更内容

IAM 系统已统一使用 **JWKS (JSON Web Key Set) 非对称签名方案**,不再依赖外部注入的 `JWT_SECRET` 对称密钥。

### 变更原因

1. **安全性提升**: 私钥仅在签发端,验证端只需公钥,降低密钥泄露风险
2. **轮换友好**: JWKS 支持多密钥并存 (kid),滚动发布时兼容旧 token
3. **集成简化**: 下游服务通过 JWKS endpoint 获取公钥,无需同步对称密钥
4. **符合标准**: 遵循 OAuth 2.0 / OpenID Connect 标准实践

### 实际影响

**系统内部已使用 JWKS (RS256)**:

- JWT 签发: `internal/apiserver/infra/jwt/generator.go` 使用 RSA 私钥签名
- JWT 验证: `pkg/sdk/authn` 通过 JWKS URL 获取公钥验证
- 密钥管理: 自动生成、轮换、持久化到 `/app/data/keys`

**本次清理仅移除遗留配置**:

- 移除 `JWT_SECRET` 环境变量要求
- 移除 `pkg/auth/auth.go` 中未使用的 HMAC 签名函数
- 更新文档和配置示例

---

## ✅ 已完成的清理工作

### 1. 代码清理

**移除遗留 HMAC 函数** (`pkg/auth/auth.go`):

```go
// 已删除的函数 (无调用者):
// - func Sign(secretID, secretKey, iss, aud string) string
// - func SignWithExpiry(...) string

// 保留的密码加密函数:
// - func Encrypt(source string) (string, error)
// - func Compare(hashedPassword, password string) error
```

### 2. 配置清理

**移除 JWT_SECRET 配置项**:

- `configs/env/config.dev.env`
- `configs/env/config.prod.env`
- `build/docker/infra/dev.env.sample`

**添加说明注释**:

```bash
# JWT 签名使用 JWKS (RS256) 非对称密钥,自动生成并持久化到 /app/data/keys
```

### 3. CI/CD 更新

**.github/workflows/cicd.yml**:

- 从必填 Secrets 列表移除 `JWT_SECRET`
- 移除环境变量引用
- 更新部署包生成逻辑

### 4. Docker Compose 增强

**开发环境** (`docker-compose.dev.yml`):

```yaml
volumes:
  - ./data/iam-keys:/app/data/keys:rw
```

**生产环境** (`docker-compose.prod.yml`):

```yaml
volumes:
  - /data/ops/iam-keys:/app/data/keys:rw
```

### 5. 环境变量打印

**移除 JWT_SECRET 打印** (`pkg/app/app.go`):

```go
keys := []string{
  // ... 其他配置
  // "IAM_APISERVER_JWT_SECRET", // 已移除
  "IAM_APISERVER_IDP_ENCRYPTION_KEY",
}
```

---

## 🚀 迁移步骤

### 对于已部署的环境

#### 1. 验证当前密钥目录

```bash
# SSH 到生产服务器
ssh user@production-server

# 检查密钥目录是否存在
ls -la /data/ops/iam-keys/

# 应该看到类似输出:
# drwxr-x--- 2 www-data www-data 4096 Dec  3 10:00 .
# -rw------- 1 www-data www-data 1679 Dec  3 10:00 key-1733200800.pem
# -rw-r--r-- 1 www-data www-data  451 Dec  3 10:00 key-1733200800.pub
```

#### 2. 验证 JWKS endpoint

```bash
# 访问 JWKS 公钥端点
curl https://iam.yourdomain.com/.well-known/jwks.json

# 应该返回 JSON:
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "key-1733200800",
      "use": "sig",
      "alg": "RS256",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

#### 3. 更新部署配置

**删除 GitHub Secrets** (可选,避免混淆):

```bash
# 在 GitHub 仓库设置中删除 JWT_SECRET
# Settings -> Secrets and variables -> Actions -> Repository secrets
# 删除: JWT_SECRET
```

**更新环境变量文件**:

```bash
# 编辑生产环境配置
vim /opt/iam-contracts/configs/env/config.prod.env

# 删除这一行:
# JWT_SECRET=xxx

# 保存后重启服务
docker-compose restart iam-apiserver
```

#### 4. 验证服务正常

```bash
# 检查容器日志
docker logs -f iam-apiserver

# 应该看到密钥加载日志:
# [INFO] JWKS: Loaded 1 active keys from /app/data/keys

# 测试登录和 Token 验证
curl -X POST https://iam.yourdomain.com/v1/authn/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"testpass"}'
```

---

## 🔍 常见问题

### Q1: 如果密钥目录不存在会怎样?

**A**: 服务启动时会自动生成新密钥并保存到 `/app/data/keys`。

```bash
# 首次启动日志:
[INFO] JWKS: No keys found, generating initial key...
[INFO] JWKS: Generated new key with kid=key-1733200800
[INFO] JWKS: Saved private key to /app/data/keys/key-1733200800.pem
[INFO] JWKS: Saved public key to /app/data/keys/key-1733200800.pub
```

### Q2: 旧 token 会失效吗?

**A**: 

- **如果之前使用 JWKS**: 不会失效,密钥持久化保证了兼容性
- **如果之前使用 JWT_SECRET**: 已经不存在这种情况,系统早已切换到 JWKS

### Q3: 下游服务需要改动吗?

**A**: 

- **如果使用 IAM SDK**: 无需改动,SDK 已支持 JWKS
- **如果自行验签**: 确保从 `/.well-known/jwks.json` 获取公钥,而非使用 JWT_SECRET

### Q4: 密钥轮换会影响服务吗?

**A**: 不会。JWKS 支持多密钥并存:

- 新 token 使用新 kid 签名
- 旧 token 仍可用旧 kid 验证
- 宽限期内 (默认 24 小时) 同时有效

### Q5: 如何手动轮换密钥?

**A**: 

```bash
# 调用密钥轮换 API (需要管理员权限)
curl -X POST https://iam.yourdomain.com/v1/admin/jwks/rotate \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# 或等待自动轮换 (默认每 30 天)
```

---

## 📚 相关文档

- [JWKS 发布指南](../modules/authn/JWKS_GUIDE.md)
- [认证中心架构](../modules/authn/README.md)
- [密钥轮换策略](../modules/authn/KEY_ROTATION.md)
- [部署指南](./deployment/README.md)

---

## 🆘 故障排查

### 服务启动失败 - 密钥目录权限错误

**错误信息**:

```
[ERROR] Failed to save private key: permission denied
```

**解决方案**:

```bash
# 修复目录权限
sudo chown -R www-data:www-data /data/ops/iam-keys
sudo chmod 750 /data/ops/iam-keys
sudo chmod 600 /data/ops/iam-keys/*.pem
sudo chmod 644 /data/ops/iam-keys/*.pub
```

### Token 验证失败 - 找不到 kid

**错误信息**:

```json
{"error":"token kid not found in JWKS"}
```

**解决方案**:

```bash
# 检查 JWKS endpoint 是否包含该 kid
curl https://iam.yourdomain.com/.well-known/jwks.json | jq '.keys[].kid'

# 如果缺失,可能是密钥已轮换,需要重新登录获取新 token
```

---

## ✨ 迁移完成检查清单

- [ ] 生产环境密钥目录挂载正常 (`/data/ops/iam-keys`)
- [ ] JWKS endpoint 可访问 (`/.well-known/jwks.json`)
- [ ] 从 GitHub Secrets 删除 JWT_SECRET (可选)
- [ ] 配置文件中移除 JWT_SECRET
- [ ] 服务启动正常,日志显示 "Loaded X active keys"
- [ ] 登录测试成功,获取 token
- [ ] Token 验证成功 (本地或通过 SDK)
- [ ] 下游服务集成验证通过

---

**迁移完成日期**: _____________  
**操作人员**: _____________  
**验证人员**: _____________
