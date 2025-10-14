package request

import (
	"encoding/json"
)

// LoginRequest 统一登录请求（符合 API 文档）
type LoginRequest struct {
	Method      string          `json:"method" binding:"required,oneof=basic wx:minip"` // 认证方式：basic 或 wx:minip
	Audience    string          `json:"audience,omitempty"`                             // web | mobile | admin
	DeviceID    string          `json:"deviceId,omitempty"`                             // 设备 ID
	Credentials json.RawMessage `json:"credentials" binding:"required"`                 // 凭证（根据 method 不同而不同）
}

// BasicCredentials 基本认证凭证（用户名密码）
type BasicCredentials struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// WeChatMiniCredentials 微信小程序凭证
type WeChatMiniCredentials struct {
	AppID  string `json:"appId" binding:"required"`
	JSCode string `json:"jsCode" binding:"required"`
}

// RefreshTokenRequest 刷新令牌请求（符合 API 文档）
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
	Audience     string `json:"audience,omitempty"` // web | mobile | admin
}

// LogoutRequest 登出请求（符合 API 文档）
type LogoutRequest struct {
	RefreshToken string `json:"refreshToken,omitempty"` // 若提供，仅撤销该票据
	All          bool   `json:"all,omitempty"`          // true 撤销当前用户所有 refresh（需鉴权）
}

// VerifyTokenRequest 验证令牌请求（符合 API 文档）
type VerifyTokenRequest struct {
	Token    string `json:"token,omitempty"`    // 可省略并使用 Authorization: Bearer 的 token
	Audience string `json:"audience,omitempty"` // 受众
}
