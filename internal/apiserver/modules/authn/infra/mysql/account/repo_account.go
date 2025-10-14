package account

import (
	"context"
	"time"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	userdomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/database/mysql"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
	"gorm.io/gorm"
)

// AccountRepository MySQL 实现。
type AccountRepository struct {
	mysql.BaseRepository[*AccountPO]
	mapper *Mapper
	db     *gorm.DB
}

var _ port.AccountRepo = (*AccountRepository)(nil)

// NewAccountRepository 创建账号仓储实现。
func NewAccountRepository(db *gorm.DB) port.AccountRepo {
	return &AccountRepository{
		BaseRepository: mysql.NewBaseRepository[*AccountPO](db),
		mapper:         NewMapper(),
		db:             db,
	}
}

// Create 创建新账号。
func (r *AccountRepository) Create(ctx context.Context, a *domain.Account) error {
	po := r.mapper.ToAccountPO(a)
	return r.BaseRepository.CreateAndSync(ctx, po, func(updated *AccountPO) {
		a.ID = domain.AccountID(updated.ID)
	})
}

// FindByID 根据主键查询账号。
func (r *AccountRepository) FindByID(ctx context.Context, id domain.AccountID) (*domain.Account, error) {
	po, err := r.BaseRepository.FindByID(ctx, idutil.ID(id).Value())
	if err != nil {
		return nil, err
	}
	acc := r.mapper.ToAccountBO(po)
	if acc == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return acc, nil
}

// FindByRef 根据外部引用查找账号。
func (r *AccountRepository) FindByRef(ctx context.Context, provider domain.Provider, externalID string, appID *string) (*domain.Account, error) {
	var po AccountPO
	q := r.db.WithContext(ctx).Where("provider = ? AND external_id = ?", string(provider), externalID)
	if appID == nil {
		q = q.Where("app_id IS NULL")
	} else {
		q = q.Where("app_id = ?", *appID)
	}
	if err := q.First(&po).Error; err != nil {
		return nil, err
	}
	acc := r.mapper.ToAccountBO(&po)
	if acc == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return acc, nil
}

// UpdateStatus 更新账号状态。
func (r *AccountRepository) UpdateStatus(ctx context.Context, id domain.AccountID, status domain.AccountStatus) error {
	return r.updateColumns(ctx, id, map[string]any{
		"status": int8(status),
	})
}

// UpdateUserID 更新账号关联的用户。
func (r *AccountRepository) UpdateUserID(ctx context.Context, id domain.AccountID, userID userdomain.UserID) error {
	return r.updateColumns(ctx, id, map[string]any{
		"user_id": idutil.ID(userID),
	})
}

// UpdateExternalRef 更新账号的外部引用信息。
func (r *AccountRepository) UpdateExternalRef(ctx context.Context, id domain.AccountID, externalID string, appID *string) error {
	updates := map[string]any{
		"external_id": externalID,
	}
	if appID == nil {
		updates["app_id"] = gorm.Expr("NULL")
	} else {
		updates["app_id"] = *appID
	}
	return r.updateColumns(ctx, id, updates)
}

func (r *AccountRepository) updateColumns(ctx context.Context, id domain.AccountID, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	now := time.Now()
	updates["updated_at"] = now
	updates["updated_by"] = idutil.NewID(0)
	updates["version"] = gorm.Expr("version + 1")

	return r.db.WithContext(ctx).
		Model(&AccountPO{}).
		Where("id = ?", idutil.ID(id).Value()).
		Updates(updates).
		Error
}
