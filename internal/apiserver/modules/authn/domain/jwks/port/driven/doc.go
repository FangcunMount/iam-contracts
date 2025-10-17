// Package driven 定义被驱动端口（Driven Ports）
//
// 被驱动端口是领域层对基础设施层的抽象，遵循依赖倒置原则。
// 领域层定义接口，基础设施层实现接口，使得领域逻辑不依赖于具体技术实现。
//
// # 端口分类
//
// 1. KeyRepository - 密钥仓储
//   - 负责 Key 实体的持久化（CRUD）
//   - 实现方：MySQL、PostgreSQL、Redis 等
//
// 2. KeyGenerator - 密钥生成器
//   - 负责生成密钥对（RSA/EC/OKP）
//   - 实现方：crypto/rsa、crypto/ecdsa、crypto/ed25519
//
// 3. PrivateKeyResolver - 私钥解析器
//   - 负责解析和获取私钥（用于签名）
//   - 实现方：PEM 文件、KMS（AWS/Azure）、HSM 等
//
// 4. KeySetReader - 密钥集读取器
//   - 负责读取当前的 JWKS（用于发布）
//   - 实现方：缓存层、聚合服务等
//
// # 依赖关系
//
// 领域服务（Domain Service）依赖这些端口接口：
//   - KeyManager 依赖 KeyRepository、KeyGenerator
//   - KeySetBuilder 依赖 KeyRepository
//   - KeyRotator 依赖 KeyRepository、KeyGenerator
//
// # 实现位置
//
// 具体实现位于 infrastructure 层：
//   - infrastructure/persistence/mysql_key_repository.go
//   - infrastructure/crypto/rsa_key_generator.go
//   - infrastructure/crypto/pem_privkey_resolver.go
package driven
