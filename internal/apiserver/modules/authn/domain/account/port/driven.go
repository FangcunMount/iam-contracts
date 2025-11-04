package port

import (
	"context"
	"time"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ==================== Driven Ports (驱动端口) ====================
// 定义领域模型、领域服务所依赖的外部服务接口，基础设施适配层需提供实现

// --------------- Repositories（端口）：账号存储库 ---------------

// 主实体：Account
type AccountRepo interface {
	// Create 创建账号
	Create(ctx context.Context, a *domain.Account) error

	// Update*** 更新账号信息
	UpdateUniqueID(ctx context.Context, id meta.ID, uniqueID domain.UnionID) error
	UpdateStatus(ctx context.Context, id meta.ID, status domain.AccountStatus) error
	UpdateProfile(ctx context.Context, id meta.ID, profile map[string]string) error
	UpdateMeta(ctx context.Context, id meta.ID, meta map[string]string) error

	// GetBy*** 查询账号
	GetByID(ctx context.Context, id meta.ID) (*domain.Account, error)
	GetByUniqueID(ctx context.Context, uniqueID domain.UnionID) (*domain.Account, error)
	GetByExternalIDAppId(ctx context.Context, externalID domain.ExternalID, appID domain.AppId) (*domain.Account, error)
}

// --------------- Credential Repository ---------------

// 主实体：Credential
type CredentialRepo interface {
	// Create 创建凭据
	Create(ctx context.Context, c *domain.Credential) error

	// Update*** 更新凭据信息
	UpdateMaterial(ctx context.Context, id meta.ID, material []byte, algo string) error
	UpdateStatus(ctx context.Context, id meta.ID, status domain.CredentialStatus) error
	UpdateFailedAttempts(ctx context.Context, id meta.ID, attempts int) error
	UpdateLockedUntil(ctx context.Context, id meta.ID, lockedUntil *time.Time) error
	UpdateLastSuccessAt(ctx context.Context, id meta.ID, lastSuccessAt time.Time) error
	UpdateLastFailureAt(ctx context.Context, id meta.ID, lastFailureAt time.Time) error
	UpdateExpiresAt(ctx context.Context, id meta.ID, expiresAt *time.Time) error

	// GetBy*** 查询凭据
	GetByID(ctx context.Context, id meta.ID) (*domain.Credential, error)
	GetByAccountIDAndType(ctx context.Context, accountID meta.ID, credType domain.CredentialType) (*domain.Credential, error)
	GetByIDPIdentifier(ctx context.Context, idpIdentifier string, credType domain.CredentialType) (*domain.Credential, error)
	ListByAccountID(ctx context.Context, accountID meta.ID) ([]*domain.Credential, error)

	// Delete 删除凭据
	Delete(ctx context.Context, id meta.ID) error
}
