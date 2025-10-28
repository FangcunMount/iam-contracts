package service

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp/port"
)

type CredentialManager struct {
	// 依赖的仓储或外部服务接口
}

// 确保 CredentialManager 实现了相应的接口
var _ port.CredentialManager = (*CredentialManager)(nil)

// NewCredentialManager 创建凭据管理器实例
func NewCredentialManager() *CredentialManager {
	return &CredentialManager{}
}

// ChangeAuthSecret 变更 AuthSecret
func (cm *CredentialManager) ChangeAuthSecret(ctx context.Context, app *domain.WechatApp, newSecret string) error {
	return nil
}

// ChangeMsgSecret 变更消息加解密密钥
func (cm *CredentialManager) ChangeMsgSecret(ctx context.Context, app *domain.WechatApp, newSecret string) error {
	return nil
}

// ChangeAPISymKey 变更 API 密钥（对称加密）
func (cm *CredentialManager) ChangeAPISymKey(ctx context.Context, app *domain.WechatApp, newKey string) error {
	return nil
}

// ChangeAPIAsymKeyWithKMS 变更 API 密钥（非对称加密）
func (cm *CredentialManager) ChangeAPIAsymKeyWithKMS(ctx context.Context, app *domain.WechatApp, newKey string) error {
	return nil
}

// ChangeAPIAsymKeyWithPlain 变更 API 密钥（明文）
func (cm *CredentialManager) ChangeAPIAsymKeyWithPlain(ctx context.Context, app *domain.WechatApp, newKey string) error {
	return nil
}
