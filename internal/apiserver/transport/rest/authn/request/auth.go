package request

import (
	"encoding/json"
	tokendomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/token"
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authn/session"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// LoginV3Request 是 v3 显式登录请求。
type LoginV3Request struct {
	AuthMethod    string          `json:"auth_method" binding:"required" enums:"password,phone_otp,wechat,wechat_scan,wecom"` // 认证方式：password | phone_otp | wechat | wechat_scan | wecom
	DeviceID      string          `json:"device_id,omitempty"`                                                                // 设备 ID
	MethodPayload json.RawMessage `json:"method_payload" binding:"required" swaggertype:"object"`                             // 凭证（wechat_scan 需要 app_id/code/state；其他方式按 auth_method 解析）
}

// Validate 验证 v3 登录请求。
func (r *LoginV3Request) Validate() error {
	if !session.IsPublicAuthMethod(r.AuthMethod) {
		return perrors.WithCode(code.ErrUnsupportedAuthMethod, "invalid authentication method: %s", r.AuthMethod)
	}
	if len(r.MethodPayload) == 0 {
		return perrors.WithCode(code.ErrPayloadInvalid, "method_payload is required")
	}
	return nil
}

// PasswordCredentials 密码认证凭证
type PasswordCredentials struct {
	// Username 登录名：须与登录身份 identifier 一致（例如配置的登录名或邮箱）
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// PhoneOTPCredentials 手机号验证码凭证
type PhoneOTPCredentials struct {
	Phone   string `json:"phone" binding:"required"`    // E.164 格式
	OTPCode string `json:"otp_code" binding:"required"` // 验证码
}

// SendLoginPhoneOTPRequest 请求发送手机登录短信验证码。
type SendLoginPhoneOTPRequest struct {
	Phone string `json:"phone" binding:"required"` // 支持 E.164 或国内手机号，服务端规范为 E.164
}

// Validate 校验发送登录 OTP 请求。
func (r *SendLoginPhoneOTPRequest) Validate() error {
	if strings.TrimSpace(r.Phone) == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "phone is required")
	}
	return nil
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
	AccessToken      string   `json:"access_token" binding:"required"`
	ExpectedIssuer   string   `json:"expected_issuer,omitempty"`
	ExpectedAudience []string `json:"expected_audience" binding:"required,min=1"`
}

// Validate 验证令牌验证请求
func (r *VerifyTokenRequest) Validate() error {
	if r.AccessToken == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "access_token is required")
	}
	audience, err := tokendomain.NormalizeExpectedAudience(r.ExpectedAudience)
	if err != nil {
		return err
	}
	r.ExpectedAudience = audience
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
