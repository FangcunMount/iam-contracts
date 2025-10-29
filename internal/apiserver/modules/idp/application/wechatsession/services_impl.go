package wechatsession

import (
	"context"
	"fmt"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession/port"
)

// ============= 应用服务实现 =============

// ===================================================
// ==== WechatAuthApplicationService 实现 =====
// ===================================================

type wechatAuthApplicationService struct {
	authenticator port.Authenticator
}

// NewWechatAuthApplicationService 创建微信认证应用服务
func NewWechatAuthApplicationService(
	authenticator port.Authenticator,
) WechatAuthApplicationService {
	return &wechatAuthApplicationService{
		authenticator: authenticator,
	}
}

// LoginWithCode 使用微信登录码进行登录
func (s *wechatAuthApplicationService) LoginWithCode(ctx context.Context, dto LoginWithCodeDTO) (*LoginResult, error) {
	// 参数校验
	if dto.AppID == "" {
		return nil, fmt.Errorf("appID cannot be empty")
	}
	if dto.JSCode == "" {
		return nil, fmt.Errorf("jsCode cannot be empty")
	}

	// 调用领域服务进行登录
	claim, session, err := s.authenticator.LoginWithCode(ctx, dto.AppID, dto.JSCode)
	if err != nil {
		return nil, fmt.Errorf("failed to login with code: %w", err)
	}

	// 转换为结果 DTO
	return toLoginResult(claim, session), nil
}

// DecryptUserPhone 解密用户手机号
func (s *wechatAuthApplicationService) DecryptUserPhone(ctx context.Context, dto DecryptPhoneDTO) (string, error) {
	// 参数校验
	if dto.AppID == "" {
		return "", fmt.Errorf("appID cannot be empty")
	}
	if dto.OpenID == "" {
		return "", fmt.Errorf("openID cannot be empty")
	}
	if dto.EncryptedData == "" {
		return "", fmt.Errorf("encryptedData cannot be empty")
	}
	if dto.IV == "" {
		return "", fmt.Errorf("iv cannot be empty")
	}

	// 调用领域服务解密手机号
	phone, err := s.authenticator.DecryptPhone(ctx, dto.AppID, dto.OpenID, dto.EncryptedData, dto.IV)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt phone: %w", err)
	}

	return phone, nil
}

// ============= 辅助函数 =============

// toLoginResult 转换领域对象为登录结果 DTO
func toLoginResult(claim *domain.ExternalClaim, session *domain.WechatSession) *LoginResult {
	if claim == nil || session == nil {
		return nil
	}

	result := &LoginResult{
		Provider:     claim.Provider,
		AppID:        claim.AppID,
		OpenID:       claim.Subject,
		UnionID:      claim.UnionID,
		DisplayName:  claim.DisplayName,
		AvatarURL:    claim.AvatarURL,
		Phone:        claim.Phone,
		Email:        claim.Email,
		ExpiresInSec: claim.ExpiresInSec,
		Version:      session.Ver,
	}

	return result
}
