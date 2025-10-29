package wechatapp

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp"
)

// ============= 应用服务接口（Driving Ports）=============

// WechatAppApplicationService 微信应用管理应用服务
type WechatAppApplicationService interface {
	// CreateApp 创建微信应用
	CreateApp(ctx context.Context, dto CreateWechatAppDTO) (*WechatAppResult, error)
	// GetApp 查询微信应用
	GetApp(ctx context.Context, appID string) (*WechatAppResult, error)
}

// WechatAppCredentialApplicationService 微信应用凭据应用服务
type WechatAppCredentialApplicationService interface {
	// RotateAuthSecret 轮换认证密钥（AppSecret）
	RotateAuthSecret(ctx context.Context, appID string, newSecret string) error
	// RotateMsgSecret 轮换消息加解密密钥
	RotateMsgSecret(ctx context.Context, appID string, callbackToken string, encodingAESKey string) error
}

// WechatAppTokenApplicationService 微信应用访问令牌应用服务
type WechatAppTokenApplicationService interface {
	// GetAccessToken 获取访问令牌（带缓存和自动刷新）
	GetAccessToken(ctx context.Context, appID string) (string, error)
	// RefreshAccessToken 强制刷新访问令牌
	RefreshAccessToken(ctx context.Context, appID string) (string, error)
}

// ============= DTOs =============

// CreateWechatAppDTO 创建微信应用 DTO
type CreateWechatAppDTO struct {
	AppID     string         // 微信应用 ID（必填）
	Name      string         // 应用名称（必填）
	Type      domain.AppType // 应用类型（必填：MiniProgram/MP）
	AppSecret string         // AppSecret（可选，创建时设置）
}

// WechatAppResult 微信应用结果 DTO
type WechatAppResult struct {
	ID     string         // 内部 ID
	AppID  string         // 微信应用 ID
	Name   string         // 应用名称
	Type   domain.AppType // 应用类型
	Status domain.Status  // 应用状态
}
