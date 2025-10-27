package adapter

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
)

// UserAdapter 用户中心适配器
// 作为防腐层(Anti-Corruption Layer),隔离 authn 模块对 uc 模块的直接依赖
// 负责在 authn.UserID 和 uc.UserID 之间进行转换和映射
// 所有跨模块的用户相关查询都通过此适配器进行
type UserAdapter interface {
	// ExistsUser 检查指定的 UserID 对应的用户是否存在
	ExistsUser(ctx context.Context, userID account.UserID) (bool, error)

	// GetUserStatus 获取用户状态
	GetUserStatus(ctx context.Context, userID account.UserID) (status string, err error)

	// IsUserActive 检查用户是否处于活跃状态
	IsUserActive(ctx context.Context, userID account.UserID) (bool, error)
}
