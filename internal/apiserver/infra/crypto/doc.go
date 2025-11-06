// Package crypto 提供加密相关的基础设施实现
//
// 本包实现了 domain/jwks/port/driven 包中定义的加密相关接口，
// 包括密钥生成器、私钥解析器等。
//
// # RSAKeyGenerator - RSA 密钥生成器
//
// RSAKeyGenerator 实现了 driven.KeyGenerator 接口，用于生成 RSA 密钥对。
//
// ## 功能特性
//
// - 支持算法：RS256, RS384, RS512
// - 支持密钥大小：2048, 4096 位（默认 2048）
// - 自动生成符合 JWK 规范的公钥
// - 自动验证生成的密钥
//
// ## 密钥大小选择
//
// - RS256/RS384: 推荐 2048 位（性能与安全的平衡）
// - RS512: 推荐 4096 位（更高的安全性）
//
// ## 算法说明
//
// - RS256: RSA Signature with SHA-256
// - RS384: RSA Signature with SHA-384
// - RS512: RSA Signature with SHA-512
//
// 区别在于哈希算法，密钥大小相同。
//
// ## 使用示例
//
//	// 创建默认生成器（2048 位）
//	generator := crypto.NewRSAKeyGenerator()
//
//	// 创建指定大小的生成器（4096 位）
//	generator := crypto.NewRSAKeyGeneratorWithSize(4096)
//
//	// 生成密钥对
//	keyPair, err := generator.GenerateKeyPair(ctx, "RS256", "unique-kid")
//	if err != nil {
//	    return err
//	}
//
//	// 使用私钥签名
//	privateKey := keyPair.PrivateKey.(*rsa.PrivateKey)
//	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
//
//	// 使用公钥 JWK 构建 JWKS
//	jwk := keyPair.PublicJWK
//
// ## 安全建议
//
// 1. 密钥大小：最小 2048 位，推荐 2048 或 4096 位
// 2. 密钥轮换：定期轮换密钥（如每 30 天）
// 3. 私钥保护：私钥应该安全存储（KMS/HSM）
// 4. 公钥发布：通过 /.well-known/jwks.json 发布公钥
//
// ## 性能考虑
//
// - 2048 位密钥生成：~50-100ms
// - 4096 位密钥生成：~500-1000ms
// - 签名性能：2048 位优于 4096 位
//
// ## RFC 标准
//
// - RFC 7517: JSON Web Key (JWK)
// - RFC 7518: JSON Web Algorithms (JWA)
// - RFC 8017: PKCS #1: RSA Cryptography Specifications
package crypto
