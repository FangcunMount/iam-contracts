// Package authentication 认证领域模型
package authentication

import "fmt"

// CredentialType 凭证类型
type CredentialType string

const (
	// CredentialTypeUsernamePassword 用户名密码凭证
	CredentialTypeUsernamePassword CredentialType = "username_password"
	// CredentialTypeWeChatCode 微信授权码凭证
	CredentialTypeWeChatCode CredentialType = "wechat_code"
	// CredentialTypeToken Token凭证（用于验证）
	CredentialTypeToken CredentialType = "bearer_token"
)

// Credential 凭证接口，代表用户提供的身份证明信息
type Credential interface {
	// Type 返回凭证类型
	Type() CredentialType
	// Validate 验证凭证格式是否有效
	Validate() error
}

// UsernamePasswordCredential 用户名密码凭证
type UsernamePasswordCredential struct {
	Username string
	Password string
}

// Type 实现 Credential 接口
func (c UsernamePasswordCredential) Type() CredentialType {
	return CredentialTypeUsernamePassword
}

// Validate 验证用户名密码格式
func (c UsernamePasswordCredential) Validate() error {
	if c.Username == "" {
		return fmt.Errorf("username is required")
	}
	if c.Password == "" {
		return fmt.Errorf("password is required")
	}
	return nil
}

// NewUsernamePasswordCredential 创建用户名密码凭证
func NewUsernamePasswordCredential(username, password string) *UsernamePasswordCredential {
	return &UsernamePasswordCredential{
		Username: username,
		Password: password,
	}
}

// WeChatCodeCredential 微信授权码凭证
type WeChatCodeCredential struct {
	Code  string // 微信授权码
	AppID string // 微信应用ID
}

// Type 实现 Credential 接口
func (c WeChatCodeCredential) Type() CredentialType {
	return CredentialTypeWeChatCode
}

// Validate 验证微信凭证格式
func (c WeChatCodeCredential) Validate() error {
	if c.Code == "" {
		return fmt.Errorf("wechat code is required")
	}
	if c.AppID == "" {
		return fmt.Errorf("wechat app_id is required")
	}
	return nil
}

// NewWeChatCodeCredential 创建微信授权码凭证
func NewWeChatCodeCredential(code, appID string) *WeChatCodeCredential {
	return &WeChatCodeCredential{
		Code:  code,
		AppID: appID,
	}
}

// TokenCredential Token凭证（用于Bearer认证）
type TokenCredential struct {
	Token string // JWT Token 字符串
}

// Type 实现 Credential 接口
func (c TokenCredential) Type() CredentialType {
	return CredentialTypeToken
}

// Validate 验证Token格式
func (c TokenCredential) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("token is required")
	}
	return nil
}

// NewTokenCredential 创建Token凭证
func NewTokenCredential(token string) *TokenCredential {
	return &TokenCredential{
		Token: token,
	}
}
