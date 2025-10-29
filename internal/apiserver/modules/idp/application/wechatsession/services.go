package wechatsession

import (
	"context"
)

// ============= 应用服务接口（Driving Ports）=============

// WechatAuthApplicationService 微信认证应用服务
type WechatAuthApplicationService interface {
	// LoginWithCode 使用微信登录码进行登录
	LoginWithCode(ctx context.Context, dto LoginWithCodeDTO) (*LoginResult, error)
	// DecryptUserPhone 解密用户手机号
	DecryptUserPhone(ctx context.Context, dto DecryptPhoneDTO) (string, error)
}

// ============= DTOs =============

// LoginWithCodeDTO 微信登录 DTO
type LoginWithCodeDTO struct {
	AppID  string // 微信应用 ID（必填）
	JSCode string // 微信登录码（必填）
}

// DecryptPhoneDTO 解密手机号 DTO
type DecryptPhoneDTO struct {
	AppID         string // 微信应用 ID（必填）
	OpenID        string // 用户 OpenID（必填）
	EncryptedData string // 加密数据（必填）
	IV            string // 加密算法的初始向量（必填）
}

// LoginResult 登录结果 DTO
type LoginResult struct {
	// 外部身份声明
	Provider     string  // 身份提供商（wechat_miniprogram）
	AppID        string  // 微信应用 ID
	OpenID       string  // 用户 OpenID
	UnionID      *string // 用户 UnionID（可选）
	DisplayName  *string // 显示名称（可选）
	AvatarURL    *string // 头像 URL（可选）
	Phone        *string // 手机号（可选）
	Email        *string // 邮箱（可选）
	ExpiresInSec int     // 过期时间（秒）

	// 会话信息
	SessionKey string // Session Key（加密后）
	Version    int    // 会话版本
}
