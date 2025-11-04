package port

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ==================== Driving Ports (驱动端口) ====================
// 这些接口由领域层（领域服务）实现，供应用层调用
// 按照功能职责拆分，遵循接口隔离原则

// --------------- Services（服务）：账号创建器 ---------------

// AccountCreater 账号创建器
type AccountCreater interface {
	Create(ctx context.Context, dto CreateAccountDTO) (*domain.Account, error)
}

// AccountEditor 账号编辑器
type AccountEditor interface {
	// SetUniqueID 设置唯一标识
	SetUniqueID(ctx context.Context, accountID meta.ID, uniqueID domain.UnionID) (*domain.Account, error)
	// UpdateProfile 更新账号资料
	UpdateProfile(ctx context.Context, accountID meta.ID, profile map[string]string) (*domain.Account, error)
	// UpdateMeta 更新账号元数据
	UpdateMeta(ctx context.Context, accountID meta.ID, meta map[string]string) (*domain.Account, error)
}

// CreateAccountDTO 创建账号数据传输对象
type CreateAccountDTO struct {
	UserID      meta.ID
	AccountType domain.AccountType
	ExternalID  domain.ExternalID
	AppID       domain.AppId
}
