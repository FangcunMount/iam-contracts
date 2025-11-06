package request

import (
	"encoding/json"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// LoginRequest 统一登录请求
type LoginRequest struct {
	Method      string          `json:"method" binding:"required"`      // 认证方式：password | phone_otp | wechat | wecom
	DeviceID    string          `json:"device_id,omitempty"`            // 设备 ID
	Credentials json.RawMessage `json:"credentials" binding:"required"` // 凭证（根据 method 不同而不同）
}

// Validate 验证登录请求
func (r *LoginRequest) Validate() error {
	validMethods := map[string]bool{
		"password":  true,
		"phone_otp": true,
		"wechat":    true,
		"wecom":     true,
	}
	if !validMethods[r.Method] {
		return perrors.WithCode(code.ErrInvalidArgument, "invalid authentication method: %s", r.Method)
	}
	if len(r.Credentials) == 0 {
		return perrors.WithCode(code.ErrInvalidArgument, "credentials is required")
	}
	return nil
}

// PasswordCredentials 密码认证凭证
type PasswordCredentials struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	TenantID uint64 `json:"tenant_id,omitempty"`
}

// PhoneOTPCredentials 手机号验证码凭证
type PhoneOTPCredentials struct {
	Phone   string `json:"phone" binding:"required"`    // E.164 格式
	OTPCode string `json:"otp_code" binding:"required"` // 验证码
}

// WeChatCredentials 微信小程序凭证
type WeChatCredentials struct {
	AppID string `json:"app_id" binding:"required"` // 微信应用ID
	Code  string `json:"code" binding:"required"`   // 微信 JS Code
}

// WeComCredentials 企业微信凭证
type WeComCredentials struct {
	CorpID   string `json:"corp_id" binding:"required"`   // 企业ID
	AuthCode string `json:"auth_code" binding:"required"` // 授权码
}

// RefreshTokenRequest 刷新令牌请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Validate 验证刷新令牌请求
func (r *RefreshTokenRequest) Validate() error {
	if r.RefreshToken == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "refresh_token is required")
	}
	return nil
}

// LogoutRequest 登出请求
type LogoutRequest struct {
	AccessToken  *string `json:"access_token,omitempty"`  // 可选，撤销访问令牌
	RefreshToken string  `json:"refresh_token,omitempty"` // 可选，撤销刷新令牌
}

// Validate 验证登出请求
func (r *LogoutRequest) Validate() error {
	if r.AccessToken == nil && r.RefreshToken == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "at least one of access_token or refresh_token is required")
	}
	return nil
}

// VerifyTokenRequest 验证令牌请求
type VerifyTokenRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

// Validate 验证令牌验证请求
func (r *VerifyTokenRequest) Validate() error {
	if r.AccessToken == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "access_token is required")
	}
	return nil
}

// RevokeTokenRequest 撤销访问令牌请求
type RevokeTokenRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

// Validate 验证撤销令牌请求
func (r *RevokeTokenRequest) Validate() error {
	if r.AccessToken == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "access_token is required")
	}
	return nil
}

// RevokeRefreshTokenRequest 撤销刷新令牌请求
type RevokeRefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Validate 验证撤销刷新令牌请求
func (r *RevokeRefreshTokenRequest) Validate() error {
	if r.RefreshToken == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "refresh_token is required")
	}
	return nil
}
