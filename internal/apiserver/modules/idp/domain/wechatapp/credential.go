package wechatapp

import (
	"time"
)

// Credentials 微信应用凭据集合
type Credentials struct {
	Auth *AuthSecret       // 登录/换 token：AppSecret
	Msg  *MsgSecret        // 消息推送“安全模式”：CallbackToken + EncodingAESKey
	API  *APISecureChannel // 接口层安全：对称加密 & 非对称签名（可选）
}

// 1) 登录/换 token
type AuthSecret struct {
	AppSecretCipher []byte // AppSecret 密文（AES-GCM/KMS 包装）
	Version         int
	LastRotatedAt   *time.Time
}

// 2) 消息推送“安全模式”
type MsgSecret struct {
	CallbackToken        string
	EncodingAESKeyCipher []byte
	Version              int
	LastRotatedAt        *time.Time
}

// 3) 接口层安全（可选）
type APISecureChannel struct {
	Sym  *SymKey  // 对称加密（加/解密报文）
	Asym *AsymKey // 非对称（签名/验签）
}

// 对称密钥
type SymKey struct {
	Alg           CryptoAlg // AES256/SM4
	KeyCipher     []byte    // 44 Base64 -> 32 bytes（密文形式）
	Version       int
	LastRotatedAt *time.Time
}

// 非对称密钥
type AsymKey struct {
	Alg           CryptoAlg // RSA/SM2
	PubPEM        []byte    // 公钥 PEM（明文可）
	PriCipher     []byte    // 私钥密文（如使用 KMS 托管可为空）
	PriKMSRef     *string   // KMS/HSM 引用（推荐）
	Version       int
	LastRotatedAt *time.Time
}
