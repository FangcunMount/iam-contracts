package authenticator

import (
	"context"

	perrors "github.com/FangcunMount/iam-contracts/pkg/errors"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	drivingPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port/driving"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// Authenticator 认证服务（策略模式编排器）
type Authenticator struct {
	authenticators []drivingPort.Authenticator // 认证器列表
}

// 确保 Authenticator 实现了 drivingPort.Authenticator 接口
var _ drivingPort.Authenticator = (*Authenticator)(nil)

// NewAuthenticator 创建认证服务
func NewAuthenticator(authenticators ...drivingPort.Authenticator) *Authenticator {
	return &Authenticator{
		authenticators: authenticators,
	}
}

// Supports 判断是否支持该凭证类型
func (s *Authenticator) Supports(credential authentication.Credential) bool {
	for _, authenticator := range s.authenticators {
		if authenticator.Supports(credential) {
			return true
		}
	}
	return false
}

// Authenticate 执行认证
// 根据凭证类型选择合适的认证器执行认证
func (s *Authenticator) Authenticate(ctx context.Context, credential authentication.Credential) (*authentication.Authentication, error) {
	// 验证凭证
	if err := credential.Validate(); err != nil {
		return nil, perrors.WrapC(err, code.ErrInvalidArgument, "invalid credential")
	}

	// 选择合适的认证器
	var selectedAuthenticator drivingPort.Authenticator
	for _, authenticator := range s.authenticators {
		if authenticator.Supports(credential) {
			selectedAuthenticator = authenticator
			break
		}
	}

	if selectedAuthenticator == nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "no authenticator supports credential type: %s", credential.Type())
	}

	// 执行认证
	auth, err := selectedAuthenticator.Authenticate(ctx, credential)
	if err != nil {
		return nil, err
	}

	return auth, nil
}

// RegisterAuthenticator 注册认证器（用于动态扩展）
func (s *Authenticator) RegisterAuthenticator(authenticator drivingPort.Authenticator) {
	s.authenticators = append(s.authenticators, authenticator)
}
