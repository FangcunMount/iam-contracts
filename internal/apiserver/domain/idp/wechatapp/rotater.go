package wechatapp

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"
)

// credentialRotater 凭据轮换器
type credentialRotater struct {
	vault SecretVault
	now   func() time.Time
}

// 确保 credentialRotater 实现了相应的接口
var _ CredentialRotater = (*credentialRotater)(nil)

// NewCredentialRotater 创建凭据轮换器实例
func NewCredentialRotater(vault SecretVault, now func() time.Time) CredentialRotater {
	if now == nil {
		now = time.Now
	}

	return &credentialRotater{
		vault: vault,
		now:   now,
	}
}

// RotateAuthSecret 轮换认证密钥
func (m *credentialRotater) RotateAuthSecret(ctx context.Context, app *WechatApp, newPlain string) error {
	// 验证参数
	if app == nil {
		return errors.New("app cannot be nil")
	}
	// AppSecret 通常为 32 位十六进制/字母数字；此处做“非空 + 长度>=16”宽松校验
	if strings.TrimSpace(newPlain) == "" || len(newPlain) < 16 {
		return errors.New("invalid app secret")
	}

	// 验证 app 状态
	if app.IsArchived() {
		return errors.New("cannot change credentials for archived app")
	}

	// 幂等：指纹相同则不变更，直接返回
	if app.Cred.Auth != nil && app.Cred.Auth.IsMatch(newPlain) {
		return nil
	}

	// 加密存储新的 AppSecret
	if m.vault == nil {
		return errors.New("missing secret vault for credential rotater")
	}

	// 加密存储新的 AppSecret
	cipher, err := m.vault.Encrypt(context.Background(), []byte(newPlain))
	if err != nil {
		return err
	}

	if app.Cred.Auth == nil {
		app.Cred.Auth = &AuthSecret{}
	}

	app.Cred.Auth.AppSecretCipher = cipher
	app.Cred.Auth.Fingerprint = fmt.Sprintf("%x", sha256.Sum256([]byte(newPlain)))
	app.Cred.Auth.Version++
	now := m.now()
	app.Cred.Auth.LastRotatedAt = &now

	return nil
}

// ChangeMsgSecret 变更消息加解密密钥
// ctx 上下文
// app 微信应用实体
// callbackToken 回调令牌
// encodingAESKey43 消息加解密密钥（43 位 Base64 字符串）
// @return 错误信息
func (m *credentialRotater) RotateMsgAESKey(ctx context.Context, app *WechatApp, callbackToken, encodingAESKey43 string) error {
	// 验证参数
	if app == nil {
		return errors.New("app cannot be nil")
	}
	// MsgSecret 通常为 43 位字母数字；此处做“非空 + 长度>=16”宽松校验
	if len(encodingAESKey43) != 43 || strings.TrimSpace(encodingAESKey43) == "" {
		return errors.New("invalid encoding aes key")
	}

	// 验证 app 状态
	if app.IsArchived() {
		return errors.New("cannot change credentials for archived app")
	}

	if m.vault == nil {
		return errors.New("missing secret vault for credential rotater")
	}

	// 加密存储新的 MsgSecret
	cipher, err := m.vault.Encrypt(context.Background(), []byte(encodingAESKey43))
	if err != nil {
		return err
	}

	if app.Cred.Msg == nil {
		app.Cred.Msg = &MsgSecret{}
	}

	app.Cred.Msg.CallbackToken = callbackToken
	app.Cred.Msg.EncodingAESKeyCipher = cipher
	app.Cred.Msg.Version++
	now := m.now()
	app.Cred.Msg.LastRotatedAt = &now

	return nil
}

// RotateAPISymKey 轮换 API 对称密钥
// ctx 上下文
// app 微信应用实体
// alg 加密算法
// base64Key Base64 编码的密钥
// @return 错误信息
func (m *credentialRotater) RotateAPISymKey(ctx context.Context, app *WechatApp, alg CryptoAlg, base64Key string) error {
	return nil
}

// RotateAPIAsymKey 轮换 API 非对称密钥
// ctx 上下文
// app 微信应用实体
// alg 加密算法
// kmsRef KMS 引用
// pubPEM 公钥 PEM 编码
// @return 错误信息
func (m *credentialRotater) RotateAPIAsymKey(ctx context.Context, app *WechatApp, alg CryptoAlg, kmsRef string, pubPEM []byte) error {
	return nil
}
