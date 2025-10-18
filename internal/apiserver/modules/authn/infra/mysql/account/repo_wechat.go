package account

import (
	"context"
	"time"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	drivenPort "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/database/mysql"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
	"gorm.io/gorm"
)

// WeChatRepository MySQL 实现。
type WeChatRepository struct {
	mysql.BaseRepository[*WeChatAccountPO]
	mapper *Mapper
	db     *gorm.DB
}

var _ drivenPort.WeChatRepo = (*WeChatRepository)(nil)

// NewWeChatRepository 构造。
func NewWeChatRepository(db *gorm.DB) drivenPort.WeChatRepo {
	return &WeChatRepository{
		BaseRepository: mysql.NewBaseRepository[*WeChatAccountPO](db),
		mapper:         NewMapper(),
		db:             db,
	}
}

// Create 创建新的微信账号。
func (r *WeChatRepository) Create(ctx context.Context, wx *domain.WeChatAccount) error {
	po := r.mapper.ToWeChatPO(wx)
	return r.BaseRepository.CreateAndSync(ctx, po, func(updated *WeChatAccountPO) {
		wx.AccountID = domain.AccountID(updated.AccountID)
	})
}

// FindByAccountID 通过账号 ID 查询微信账号。
func (r *WeChatRepository) FindByAccountID(ctx context.Context, accountID domain.AccountID) (*domain.WeChatAccount, error) {
	var po WeChatAccountPO
	err := r.db.WithContext(ctx).
		Where("account_id = ?", idutil.ID(accountID).Uint64()).
		First(&po).
		Error
	if err != nil {
		return nil, err
	}
	wx := r.mapper.ToWeChatBO(&po)
	if wx == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return wx, nil
}

// FindByAppOpenID 根据 AppID 与 OpenID 查询微信账号。
func (r *WeChatRepository) FindByAppOpenID(ctx context.Context, appID, openid string) (*domain.WeChatAccount, error) {
	var po WeChatAccountPO
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND open_id = ?", appID, openid).
		First(&po).
		Error
	if err != nil {
		return nil, err
	}
	wx := r.mapper.ToWeChatBO(&po)
	if wx == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return wx, nil
}

// UpdateProfile 更新昵称、头像、Meta。
func (r *WeChatRepository) UpdateProfile(ctx context.Context, accountID domain.AccountID, nickname, avatar *string, meta []byte) error {
	updates := make(map[string]any)
	if nickname != nil {
		updates["nickname"] = *nickname
	}
	if avatar != nil {
		updates["avatar_url"] = *avatar
	}
	if meta != nil {
		updates["meta"] = meta
	}
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = time.Now()
	updates["updated_by"] = idutil.NewID(0)
	updates["version"] = gorm.Expr("version + 1")

	return r.db.WithContext(ctx).
		Model(&WeChatAccountPO{}).
		Where("account_id = ?", idutil.ID(accountID).Uint64()).
		Updates(updates).
		Error
}

// UpdateUnionID 更新 UnionID。
func (r *WeChatRepository) UpdateUnionID(ctx context.Context, accountID domain.AccountID, unionID string) error {
	updates := map[string]any{
		"union_id":   unionID,
		"updated_at": time.Now(),
		"updated_by": idutil.NewID(0),
		"version":    gorm.Expr("version + 1"),
	}
	return r.db.WithContext(ctx).
		Model(&WeChatAccountPO{}).
		Where("account_id = ?", idutil.ID(accountID).Uint64()).
		Updates(updates).
		Error
}
