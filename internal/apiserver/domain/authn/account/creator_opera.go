package account

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// ==================== 运营账户创建策略 ====================

// OperaCreatorStrategy 运营账户创建策略（TypeOpera）
// 运营账户可以绑定多种凭据：密码、手机OTP等
type OperaCreatorStrategy struct{}

var _ CreatorStrategy = (*OperaCreatorStrategy)(nil)

// NewOperaCreatorStrategy 创建运营账户创建策略
func NewOperaCreatorStrategy() *OperaCreatorStrategy {
	return &OperaCreatorStrategy{}
}

// Kind 返回策略支持的账户类型
func (s *OperaCreatorStrategy) Kind() AccountType {
	return TypeOpera
}

// PrepareData 准备运营账户创建参数
func (s *OperaCreatorStrategy) PrepareData(ctx context.Context, input CreationInput) (*CreationParams, error) {
	// 验证必要参数
	if input.Phone.IsEmpty() {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "phone is required for opera account")
	}

	// 运营账户使用手机号作为 ExternalID
	externalID := ExternalID(input.Phone.String())
	appID := AppId("opera")

	return &CreationParams{
		UserID:      input.UserID,
		AccountType: TypeOpera,
		AppID:       appID,
		ExternalID:  externalID,
		Profile:     input.Profile,
		Meta:        input.Meta,
		ParamsJSON:  input.ParamsJSON,
	}, nil
}

// Create 创建运营账户实体
func (s *OperaCreatorStrategy) Create(ctx context.Context, params *CreationParams) (*Account, error) {
	// 创建账户实体
	account := NewAccount(
		params.UserID,
		TypeOpera,
		params.ExternalID,
		WithAppID(params.AppID),
	)

	// 设置资料和元数据
	if len(params.Profile) > 0 {
		account.Profile = params.Profile
	}
	if len(params.Meta) > 0 {
		account.Meta = params.Meta
	}

	return account, nil
}
