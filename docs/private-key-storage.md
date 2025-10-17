# 私钥持久化方案设计文档

## 概述

本文档描述了 JWKS 系统中 RSA 私钥的持久化方案。该方案确保在密钥生成后，私钥能够安全地保存到存储系统中，以便后续用于 JWT 签名。

## 架构设计

### 核心接口

#### PrivateKeyStorage 接口

```go
type PrivateKeyStorage interface {
    // SavePrivateKey 保存私钥
    SavePrivateKey(ctx context.Context, kid string, privateKey any, alg string) error
    
    // DeletePrivateKey 删除私钥
    DeletePrivateKey(ctx context.Context, kid string) error
    
    // KeyExists 检查私钥是否存在
    KeyExists(ctx context.Context, kid string) (bool, error)
}
```

**位置**: `domain/jwks/port/driven/private_key_storage.go`

### 实现类

#### 1. PEMPrivateKeyStorage（文件存储）

基于 PEM 文件格式的私钥存储实现，适用于开发环境和简单场景。

**特性**:
- 使用 PKCS#8 格式编码私钥
- 文件命名规则：`{kid}.pem`
- 默认文件权限：`0600`（仅所有者可读写）
- 自动创建存储目录

**位置**: `infra/crypto/pem_storage.go`

**使用示例**:

```go
// 创建存储
storage := crypto.NewPEMPrivateKeyStorage("/path/to/keys")

// 保存私钥
err := storage.SavePrivateKey(ctx, kid, privateKey, "RS256")

// 检查私钥是否存在
exists, err := storage.KeyExists(ctx, kid)

// 删除私钥
err := storage.DeletePrivateKey(ctx, kid)

// 列出所有密钥
kids, err := storage.ListKeys(ctx)
```

#### 2. RSAKeyGeneratorWithStorage（带持久化的生成器）

封装了 RSAKeyGenerator 并在生成密钥后自动保存私钥。

**特性**:
- 生成密钥对后立即持久化私钥
- 透明集成到现有系统中
- 支持不同密钥大小（2048/4096 位）

**位置**: `infra/crypto/rsa_generator_with_storage.go`

**使用示例**:

```go
// 创建带持久化的生成器
storage := crypto.NewPEMPrivateKeyStorage("/path/to/keys")
generator := crypto.NewRSAKeyGeneratorWithStorage(storage)

// 生成密钥（私钥自动保存）
keyPair, err := generator.GenerateKeyPair(ctx, "RS256", kid)
// 此时私钥已经保存到 /path/to/keys/{kid}.pem
```

## 工作流程

### 密钥生成与持久化流程

```
┌─────────────────┐
│ KeyManager      │
│ CreateKey()     │
└────────┬────────┘
         │
         ▼
┌────────────────────────────┐
│ RSAKeyGeneratorWithStorage │
│ GenerateKeyPair()          │
└────────┬───────────────────┘
         │
         ├──► 1. 生成 RSA 密钥对
         │     ├─ PrivateKey (*rsa.PrivateKey)
         │     └─ PublicJWK (JWK format)
         │
         ├──► 2. 保存私钥到存储
         │     PrivateKeyStorage.SavePrivateKey()
         │     └─ /keys/{kid}.pem (PKCS#8 PEM)
         │
         └──► 3. 返回密钥对
               └─ KeyManager 保存公钥到数据库
```

### JWT 签名流程

```
┌─────────────┐
│ Generator   │
│ SignJWT()   │
└──────┬──────┘
       │
       ├──► 1. 获取活跃密钥 (GetActiveKey)
       │     └─ kid, PublicJWK (from Database)
       │
       ├──► 2. 解析私钥
       │     PEMPrivateKeyResolver.ResolveSigningKey()
       │     └─ 读取 /keys/{kid}.pem
       │     └─ 解析为 *rsa.PrivateKey
       │
       └──► 3. 签名 JWT
             jwt.SignedString(privateKey)
             └─ 返回带签名的 JWT
```

## 安全考虑

### 文件权限

- **默认权限**: `0600` (rw-------)
- **目录权限**: `0755` (rwxr-xr-x)
- **建议**: 生产环境应使用更严格的权限控制

### 文件命名

- **格式**: `{kid}.pem`
- **kid**: UUID v4 格式，保证唯一性
- **示例**: `f78c0749-b3ab-4550-af77-6423a7af77ba.pem`

### 私钥格式

- **编码**: PKCS#8
- **PEM 标记**: `-----BEGIN PRIVATE KEY-----`
- **优势**: 通用性强，兼容各种工具

## 生产环境部署

### 配置参数

在 `apiserver.yaml` 中配置：

```yaml
jwks:
  keys_dir: "/var/lib/iam/jwks/keys"  # 私钥存储目录
```

### 目录准备

```bash
# 创建密钥存储目录
sudo mkdir -p /var/lib/iam/jwks/keys

# 设置权限（仅 IAM 服务用户可访问）
sudo chown iam-service:iam-service /var/lib/iam/jwks/keys
sudo chmod 700 /var/lib/iam/jwks/keys
```

### 备份策略

```bash
# 定期备份密钥文件
0 2 * * * tar -czf /backup/jwks-keys-$(date +\%Y\%m\%d).tar.gz \
          /var/lib/iam/jwks/keys/
```

## 密钥清理

### 自动清理流程

```go
// 1. 查找已过期的密钥
expiredKeys, err := keyManager.FindExpired(ctx)

// 2. 删除数据库记录
for _, key := range expiredKeys {
    keyRepo.Delete(ctx, key.Kid)
}

// 3. 删除私钥文件
for _, key := range expiredKeys {
    privateKeyStorage.DeletePrivateKey(ctx, key.Kid)
}
```

### 手动清理

```bash
# 列出所有密钥文件
ls -lh /var/lib/iam/jwks/keys/

# 删除特定密钥
rm /var/lib/iam/jwks/keys/{kid}.pem
```

## 未来扩展

### KMS（密钥管理服务）

生产环境建议使用云服务提供商的 KMS：

```go
// AWS KMS 实现示例
type AWSKMSPrivateKeyStorage struct {
    kmsClient *kms.Client
}

func (s *AWSKMSPrivateKeyStorage) SavePrivateKey(ctx context.Context, kid string, privateKey any, alg string) error {
    // 使用 KMS 加密并存储私钥
    encrypted, err := s.kmsClient.Encrypt(ctx, privateKeyBytes)
    // 存储加密后的私钥到 S3 或数据库
    return err
}
```

### HSM（硬件安全模块）

高安全场景可使用 HSM：

```go
// HSM 实现示例
type HSMPrivateKeyStorage struct {
    hsmClient *pkcs11.Ctx
}

func (s *HSMPrivateKeyStorage) SavePrivateKey(ctx context.Context, kid string, privateKey any, alg string) error {
    // 将私钥导入到 HSM 设备
    return s.hsmClient.ImportKey(privateKey)
}
```

## 测试验证

### 单元测试

```bash
# 运行私钥存储测试
go test -v -run TestPEMPrivateKeyStorage ./internal/.../crypto/

# 运行带持久化的生成器测试
go test -v -run TestRSAKeyGeneratorWithStorage ./internal/.../crypto/
```

### 集成测试

```bash
# 运行生产环境 E2E 测试
go test -v -run TestE2E_Production ./internal/.../authn/
```

### 测试结果

- ✅ 私钥保存和读取
- ✅ 文件权限设置
- ✅ 密钥删除
- ✅ 列出密钥文件
- ✅ 与解析器兼容性
- ✅ 完整的签名→验证流程

## 性能考虑

### 文件 I/O 优化

- **缓存**: 考虑在内存中缓存常用私钥
- **异步写入**: 密钥生成后异步保存
- **批量操作**: 批量清理过期密钥

### 监控指标

```go
// 建议监控的指标
- private_key_save_duration_seconds
- private_key_read_duration_seconds
- private_key_file_count
- private_key_storage_errors_total
```

## 参考资料

- [RFC 7517 - JSON Web Key (JWK)](https://tools.ietf.org/html/rfc7517)
- [RFC 5208 - PKCS #8: Private-Key Information Syntax](https://tools.ietf.org/html/rfc5208)
- [OWASP Cryptographic Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html)

## 总结

本方案通过 `PrivateKeyStorage` 接口和 `RSAKeyGeneratorWithStorage` 实现了私钥的自动持久化，确保了：

1. **安全性**: 严格的文件权限控制
2. **可靠性**: 自动保存，避免私钥丢失
3. **可扩展性**: 接口化设计，易于切换到 KMS/HSM
4. **易用性**: 透明集成，开发者无需关心持久化细节

生产环境部署时，建议根据实际安全需求选择合适的存储方案（文件/KMS/HSM）。
