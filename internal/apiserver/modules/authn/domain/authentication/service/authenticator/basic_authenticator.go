package authenticator

import (
	"context"

	perrors "github.com/FangcunMount/iam-contracts/pkg/errors"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	accountDrivenPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	drivenPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// BasicAuthenticator 基础认证器（用户名密码认证）
type BasicAuthenticator struct {
	accountRepo   accountDrivenPort.AccountRepo   // 账号仓储
	operationRepo accountDrivenPort.OperationRepo // 运营账号仓储
	passwordPort  drivenPort.AccountPasswordPort  // 密码端口
}

// NewBasicAuthenticator 创建基础认证器
func NewBasicAuthenticator(
	accountRepo accountDrivenPort.AccountRepo,
	operationRepo accountDrivenPort.OperationRepo,
	passwordPort drivenPort.AccountPasswordPort,
) *BasicAuthenticator {
	return &BasicAuthenticator{
		accountRepo:   accountRepo,
		operationRepo: operationRepo,
		passwordPort:  passwordPort,
	}
}

// Supports 判断是否支持该凭证类型
func (a *BasicAuthenticator) Supports(credential authentication.Credential) bool {
	return credential.Type() == authentication.CredentialTypeUsernamePassword
}

// Authenticate 执行认证
func (a *BasicAuthenticator) Authenticate(ctx context.Context, credential authentication.Credential) (*authentication.Authentication, error) {
	// 类型断言
	upCred, ok := credential.(*authentication.UsernamePasswordCredential)
	if !ok {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "invalid credential type for basic authenticator")
	}

	// 验证凭证格式
	if err := upCred.Validate(); err != nil {
		return nil, perrors.WrapC(err, code.ErrInvalidArgument, "credential validation failed")
	}

	// 根据用户名查找运营账号
	opAccount, err := a.operationRepo.FindByUsername(ctx, upCred.Username)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrUnauthenticated, "invalid credentials")
	}

	// 获取对应的 Account
	acc, err := a.accountRepo.FindByID(ctx, opAccount.AccountID)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrUnauthenticated, "invalid credentials")
	}

	// 检查账号状态
	if acc.Status != account.StatusActive {
		return nil, perrors.WithCode(code.ErrUnauthenticated, "account is not active")
	}

	// 获取密码哈希
	passwordHash, err := a.passwordPort.GetPasswordHash(ctx, acc.ID)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "failed to get password hash")
	}

	// 验证密码
	matched, err := passwordHash.Verify(upCred.Password)
	if err != nil {
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "password verification failed")
	}
	if !matched {
		return nil, perrors.WithCode(code.ErrUnauthenticated, "invalid credentials")
	}

	// 创建认证结果
	auth := authentication.NewAuthentication(
		acc.UserID,
		acc.ID,
		acc.Provider,
		map[string]string{
			"username": upCred.Username,
		},
	)

	return auth, nil
}
