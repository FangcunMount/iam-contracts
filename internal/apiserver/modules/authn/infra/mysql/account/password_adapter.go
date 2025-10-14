// Package account 账号密码适配器
package account

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	accountPort "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
)

// PasswordAdapter 账号密码适配器
//
// 实现 authentication.port.AccountPasswordPort 接口
// 从 MySQL 数据库查询账号的密码哈希
type PasswordAdapter struct {
	operationRepo accountPort.OperationRepo
}

// NewPasswordAdapter 创建密码适配器
func NewPasswordAdapter(operationRepo accountPort.OperationRepo) *PasswordAdapter {
	return &PasswordAdapter{
		operationRepo: operationRepo,
	}
}

// GetPasswordHash 获取账号的密码哈希
func (a *PasswordAdapter) GetPasswordHash(ctx context.Context, accountID account.AccountID) (*authentication.PasswordHash, error) {
	// 查询运营账号
	opAccount, err := a.operationRepo.FindByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// 将数据库中的密码哈希转换为领域模型
	passwordHash := authentication.NewPasswordHash(
		string(opAccount.PasswordHash),
		authentication.PasswordHashAlgorithm(opAccount.Algo),
		parseHashParameters(opAccount.Params),
	)

	return passwordHash, nil
}

// parseHashParameters 解析哈希参数
func parseHashParameters(params []byte) map[string]string {
	// 如果参数为空，返回空 map
	if len(params) == 0 {
		return make(map[string]string)
	}

	// TODO: 如果需要存储更复杂的参数，可以使用 JSON 反序列化
	// 目前 Bcrypt 不需要额外参数
	return make(map[string]string)
}
