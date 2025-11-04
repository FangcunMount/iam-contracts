package port

import (
	"context"
	"time"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ==================== Driving Ports (驱动端口) ====================
// 这些接口由领域层（领域服务）实现，供应用层调用
// 按照功能职责拆分，遵循接口隔离原则

// --------------- Account 领域服务 ---------------

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

// AccountStateMachine 账号状态机
type AccountStateMachine interface {
	// Activate 激活账号
	Activate() error
	// Disable 禁用账号
	Disable() error
	// Archive 归档账号
	Archive() error
	// Delete 删除账号
	Delete() error
	// Status 获取当前状态
	Status() domain.AccountStatus
	// Account 获取当前账号对象
	Account() *domain.Account
}

// CreateAccountDTO 创建账号数据传输对象
type CreateAccountDTO struct {
	UserID      meta.ID
	AccountType domain.AccountType
	ExternalID  domain.ExternalID
	AppID       domain.AppId
}

// ------------------------------------------

// ------------ Credential 领域服务 ---------------
// 1) 绑定/创建（注册或后续绑定）
type CredentialBinder interface {
	Bind(spec BindSpec) (*domain.Credential, error)
}
type BindSpec struct {
	AccountID     int64
	Type          domain.CredentialType
	IDP           *string
	IDPIdentifier string
	AppID         *string
	Material      []byte  // 仅 password
	Algo          *string // 仅 password
	ParamsJSON    []byte
}

// 2) 使用记录/通用状态迁移（不做“如何验证”）
type CredentialUsage interface {
	EnsureUsable(c *domain.Credential, now time.Time) error // Active 且未锁
	RecordSuccess(c *domain.Credential, now time.Time)      // 归零计数
	RecordFailure(c *domain.Credential, now time.Time, p domain.LockoutPolicy) (locked bool)
}

// 3) 锁定管理（行政动作，主要给 password）
type CredentialLocker interface {
	LockUntil(c *domain.Credential, until time.Time)
	Unlock(c *domain.Credential)
}

// 4) 材料轮换（仅 password 的条件再哈希）
type CredentialRotator interface {
	Rotate(c *domain.Credential, newMaterial []byte, newAlgo *string)
}

// 5) 生命周期（启停）
type CredentialLifecycle interface {
	Enable(c *domain.Credential)
	Disable(c *domain.Credential)
}
