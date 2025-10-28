package service

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession/port"
)

type Authenticator struct {
	// 依赖的仓储或外部服务接口
}

// 确保 Authenticator 实现了相应的接口
var _ port.Authenticator = (*Authenticator)(nil)

// NewAuthenticator 创建认证器实例
func NewAuthenticator() *Authenticator {
	return &Authenticator{}
}

// LoginWithCode 使用登录码进行登录，返回统一声明和微信会话
func (a *Authenticator) LoginWithCode(ctx context.Context, appID string, jsCode string) (*domain.ExternalClaim, *domain.WechatSession, error) {
	return nil, nil, nil
}

// DecryptPhone 解密用户手机号
func (a *Authenticator) DecryptPhone(ctx context.Context, appID, openID, encryptedData, iv string) (string, error) {
	return "", nil
}
