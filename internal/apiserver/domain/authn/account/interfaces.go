package account

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ==================== Driving Ports (驱动端口) ====================
// 这些接口由领域层实现，供应用层调用
// 遵循接口隔离原则，按职责细分

// Creator 账号创建器（Driving Port）
// 职责：创建新账号
type Creator interface {
	Create(ctx context.Context, dto CreateAccountDTO) (*Account, error)
}

// Editor 账号编辑器（Driving Port）
// 职责：编辑账号信息
type Editor interface {
	// SetUniqueID 设置全局唯一标识
	SetUniqueID(ctx context.Context, accountID meta.ID, uniqueID UnionID) (*Account, error)
	// UpdateProfile 更新账号资料
	UpdateProfile(ctx context.Context, accountID meta.ID, profile map[string]string) (*Account, error)
	// UpdateMeta 更新账号元数据
	UpdateMeta(ctx context.Context, accountID meta.ID, meta map[string]string) (*Account, error)
}

// StatusManager 账号状态管理器（Driving Port）
// 职责：管理账号状态转换
type StatusManager interface {
	// Activate 激活账号
	Activate(ctx context.Context, accountID meta.ID) (*Account, error)
	// Disable 禁用账号
	Disable(ctx context.Context, accountID meta.ID) (*Account, error)
	// Archive 归档账号
	Archive(ctx context.Context, accountID meta.ID) (*Account, error)
	// Delete 删除账号
	Delete(ctx context.Context, accountID meta.ID) (*Account, error)
}
