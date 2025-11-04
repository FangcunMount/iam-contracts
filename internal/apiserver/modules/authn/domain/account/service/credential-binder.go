package service

import (
	"github.com/FangcunMount/component-base/pkg/errors"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// CredentialBinder 凭据绑定器
// 职责：将外部认证信息绑定到账号，创建凭据实体
type CredentialBinder struct{}

// Ensure CredentialBinder implements the port.CredentialBinder interface
var _ port.CredentialBinder = (*CredentialBinder)(nil)

// NewCredentialBinder creates a new CredentialBinder instance
func NewCredentialBinder() *CredentialBinder {
	return &CredentialBinder{}
}

// Bind 绑定凭据到账号
// 根据规范创建凭据实体，不涉及持久化
func (cb *CredentialBinder) Bind(spec port.BindSpec) (*domain.Credential, error) {
	// 参数校验
	if spec.AccountID == 0 {
		return nil, errors.WithCode(code.ErrInvalidArgument, "account ID cannot be zero")
	}
	if spec.Type == "" {
		return nil, errors.WithCode(code.ErrInvalidArgument, "credential type cannot be empty")
	}

	// 创建凭据实体
	cred := &domain.Credential{
		AccountID:     spec.AccountID,
		IDP:           spec.IDP,
		IDPIdentifier: spec.IDPIdentifier,
		AppID:         spec.AppID,
		Material:      spec.Material,
		Algo:          spec.Algo,
		ParamsJSON:    spec.ParamsJSON,
		Status:        domain.CredStatusEnabled, // 默认启用
	}

	// 根据类型验证必需字段
	switch spec.Type {
	case domain.CredPassword:
		// password 类型需要 Material 和 Algo
		if len(spec.Material) == 0 {
			return nil, errors.WithCode(code.ErrInvalidCredential, "password credential requires material")
		}
		if spec.Algo == nil || *spec.Algo == "" {
			return nil, errors.WithCode(code.ErrInvalidCredential, "password credential requires algo")
		}
		// password 类型不需要 IDP
		cred.IDP = nil
		cred.AppID = nil

	case domain.CredPhoneOTP:
		// phone_otp 需要 IDPIdentifier（手机号）
		if spec.IDPIdentifier == "" {
			return nil, errors.WithCode(code.ErrInvalidCredential, "phone_otp credential requires IDP identifier (phone number)")
		}
		// phone_otp 不需要 Material
		cred.Material = nil
		cred.Algo = nil

	case domain.CredOAuthWxMinip, domain.CredOAuthWecom:
		// OAuth 类型需要 IDPIdentifier 和 AppID
		if spec.IDPIdentifier == "" {
			return nil, errors.WithCode(code.ErrInvalidCredential, "OAuth credential requires IDP identifier")
		}
		if spec.AppID == nil || *spec.AppID == "" {
			return nil, errors.WithCode(code.ErrInvalidCredential, "OAuth credential requires AppID")
		}
		// OAuth 不需要 Material
		cred.Material = nil
		cred.Algo = nil

	default:
		return nil, errors.WithCode(code.ErrInvalidCredential, "unsupported credential type: %s", spec.Type)
	}

	return cred, nil
}
