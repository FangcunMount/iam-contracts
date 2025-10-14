package port

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
)

// AccountPasswordPort 账号密码端口
//
// 用于获取账号的密码哈希信息，供 BasicAuthenticator 验证密码
type AccountPasswordPort interface {
	// GetPasswordHash 获取账号的密码哈希
	GetPasswordHash(ctx context.Context, accountID account.AccountID) (*authentication.PasswordHash, error)
}
