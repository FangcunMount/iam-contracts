package account

import (
	"context"
	"time"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/database/mysql"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
	"gorm.io/gorm"
)

// OperationRepository MySQL 实现。
type OperationRepository struct {
	mysql.BaseRepository[*OperationAccountPO]
	mapper *Mapper
	db     *gorm.DB
}

var _ port.OperationRepo = (*OperationRepository)(nil)

// NewOperationRepository 创建运营账号仓储。
func NewOperationRepository(db *gorm.DB) port.OperationRepo {
	return &OperationRepository{
		BaseRepository: mysql.NewBaseRepository[*OperationAccountPO](db),
		mapper:         NewMapper(),
		db:             db,
	}
}

// Create 创建运营后台账号。
func (r *OperationRepository) Create(ctx context.Context, cred *domain.OperationAccount) error {
	po := r.mapper.ToOperationPO(cred)
	return r.BaseRepository.CreateAndSync(ctx, po, func(updated *OperationAccountPO) {
		cred.AccountID = domain.AccountID(updated.AccountID)
	})
}

// FindByAccountID 根据账号 ID 查询运营账号凭证。
func (r *OperationRepository) FindByAccountID(ctx context.Context, accountID domain.AccountID) (*domain.OperationAccount, error) {
	var po OperationAccountPO
	err := r.db.WithContext(ctx).
		Where("account_id = ?", idutil.ID(accountID).Value()).
		First(&po).
		Error
	if err != nil {
		return nil, err
	}
	cred := r.mapper.ToOperationBO(&po)
	if cred == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return cred, nil
}

// FindByUsername 根据用户名查询运营账号凭证。
func (r *OperationRepository) FindByUsername(ctx context.Context, username string) (*domain.OperationAccount, error) {
	var po OperationAccountPO
	err := r.db.WithContext(ctx).
		Where("username = ?", username).
		First(&po).
		Error
	if err != nil {
		return nil, err
	}
	cred := r.mapper.ToOperationBO(&po)
	if cred == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return cred, nil
}

// UpdateHash 更新密码哈希与算法信息。
func (r *OperationRepository) UpdateHash(ctx context.Context, username string, hash []byte, algo string, params []byte) error {
	return r.db.WithContext(ctx).
		Model(&OperationAccountPO{}).
		Where("username = ?", username).
		Updates(map[string]any{
			"password_hash":   cloneBytes(hash),
			"algo":            algo,
			"params":          cloneBytes(params),
			"failed_attempts": 0,
			"locked_until":    nil,
			"last_changed_at": time.Now(),
		}).
		Error
}

// UpdateUsername 更新用户名。
func (r *OperationRepository) UpdateUsername(ctx context.Context, accountID domain.AccountID, newUsername string) error {
	return r.db.WithContext(ctx).
		Model(&OperationAccountPO{}).
		Where("account_id = ?", idutil.ID(accountID).Value()).
		Update("username", newUsername).
		Error
}

// ResetFailures 清零失败次数。
func (r *OperationRepository) ResetFailures(ctx context.Context, username string) error {
	return r.db.WithContext(ctx).
		Model(&OperationAccountPO{}).
		Where("username = ?", username).
		Updates(map[string]any{
			"failed_attempts": 0,
			"updated_at":      time.Now(),
			"updated_by":      idutil.NewID(0),
		}).
		Error
}

// Unlock 解锁账号。
func (r *OperationRepository) Unlock(ctx context.Context, username string) error {
	return r.db.WithContext(ctx).
		Model(&OperationAccountPO{}).
		Where("username = ?", username).
		Updates(map[string]any{
			"locked_until": nil,
			"updated_at":   time.Now(),
			"updated_by":   idutil.NewID(0),
		}).
		Error
}
