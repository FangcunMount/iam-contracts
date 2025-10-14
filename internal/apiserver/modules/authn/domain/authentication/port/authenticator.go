// Package port 认证领域端口定义
package port

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
)

// Authenticator 认证器接口（策略模式）
//
// 不同的认证方式实现此接口，如：
// - BasicAuthenticator: 用户名密码认证
// - WeChatAuthenticator: 微信 OAuth 认证
// - BearerAuthenticator: Token 认证
type Authenticator interface {
	// Supports 判断是否支持该凭证类型
	Supports(credential authentication.Credential) bool

	// Authenticate 执行认证，返回认证结果
	Authenticate(ctx context.Context, credential authentication.Credential) (*authentication.Authentication, error)
}
