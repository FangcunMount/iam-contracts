package proof

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v4/internal/apiserver/application/authn/signin/method"
	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/authentication"
	"github.com/FangcunMount/iam/v4/internal/pkg/code"
)

// passwordBuilder 密码登录方式构造器
type passwordBuilder struct{}

// NewPasswordBuilder 创建密码登录方式构造器
func NewPasswordBuilder() Builder {
	return passwordBuilder{}
}

// CredentialKind 返回认证证明类型
func (passwordBuilder) CredentialKind() method.CredentialKind {
	return method.CredentialKindPassword
}

// Build 构建密码登录方式
func (passwordBuilder) Build(_ context.Context, payload method.Payload, common method.CommonPayload) (authentication.IdentityProof, error) {
	// 验证密码登录方式凭证是否有效
	passwordPayload, ok := payload.(method.PasswordPayload)
	if !ok {
		return nil, perrors.WithCode(code.ErrProofBuildFailed, "invalid password payload")
	}

	// 构建密码登录方式凭证
	return authentication.NewPasswordProof(authentication.PasswordProofSpec{
		RealmTenantID: common.TenantID,
		Username:      passwordPayload.Username,
		Password:      passwordPayload.Password,
	})
}
