// Package driven 认证领域被驱动端口定义
package driven

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
)

// AccountPasswordPort 账号密码端口
//
// 用于获取账号的密码哈希信息，供 BasicAuthenticator 验证密码
type AccountPasswordPort interface {
	// GetPasswordHash 获取账号的密码哈希
	GetPasswordHash(ctx context.Context, accountID account.AccountID) (*authentication.PasswordHash, error)
}
