package authenticator

import (
	"context"

	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	accountPort "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
)

// WeChatAuthenticator 微信认证器
type WeChatAuthenticator struct {
	accountRepo accountPort.AccountRepo // 账号仓储
	wechatRepo  accountPort.WeChatRepo  // 微信账号仓储
	wechatPort  port.WeChatAuthPort     // 微信认证端口
}

// NewWeChatAuthenticator 创建微信认证器
func NewWeChatAuthenticator(
	accountRepo accountPort.AccountRepo,
	wechatRepo accountPort.WeChatRepo,
	wechatPort port.WeChatAuthPort,
) *WeChatAuthenticator {
	return &WeChatAuthenticator{
		accountRepo: accountRepo,
		wechatRepo:  wechatRepo,
		wechatPort:  wechatPort,
	}
}

// Supports 判断是否支持该凭证类型
func (a *WeChatAuthenticator) Supports(credential authentication.Credential) bool {
	return credential.Type() == authentication.CredentialTypeWeChatCode
}

// Authenticate 执行认证
func (a *WeChatAuthenticator) Authenticate(ctx context.Context, credential authentication.Credential) (*authentication.Authentication, error) {
	// 类型断言
	wxCred, ok := credential.(*authentication.WeChatCodeCredential)
	if !ok {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "invalid credential type for wechat authenticator")
	}

	// 验证凭证格式
	if err := wxCred.Validate(); err != nil {
		return nil, perrors.WrapC(err, code.ErrInvalidArgument, "credential validation failed")
	}

	// 通过微信 code 换取 openID
	openID, err := a.wechatPort.ExchangeOpenID(ctx, wxCred.Code, wxCred.AppID)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrUnauthenticated, "failed to exchange openID from wechat")
	}

	// 根据 openID 查找微信账号
	wxAccount, err := a.wechatRepo.FindByAppOpenID(ctx, wxCred.AppID, openID)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrUnauthenticated, "wechat account not found")
	}

	// 获取对应的 Account
	acc, err := a.accountRepo.FindByID(ctx, wxAccount.AccountID)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrUnauthenticated, "account not found")
	}

	// 检查账号状态
	if acc.Status != account.StatusActive {
		return nil, perrors.WithCode(code.ErrUnauthenticated, "account is not active")
	}

	// 创建认证结果
	auth := authentication.NewAuthentication(
		acc.UserID,
		acc.ID,
		acc.Provider,
		map[string]string{
			"openid": openID,
			"app_id": wxCred.AppID,
		},
	)

	return auth, nil
}
