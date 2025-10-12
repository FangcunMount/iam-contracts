package user

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	port "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user/port/repo"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/infra/mysql"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
	"gorm.io/gorm"
)

// Repository 用户存储库实现
type Repository struct {
	mysql.BaseRepository[*UserPO]
	mapper *UserMapper
}

// NewRepository 创建用户存储库
func NewRepository(db *gorm.DB) port.UserRepository {
	return &Repository{
		BaseRepository: mysql.NewBaseRepository[*UserPO](db),
		mapper:         NewUserMapper(),
	}
}

// Create 创建新用户
func (r *Repository) Create(ctx context.Context, u user.User) error {
	po := r.mapper.ToPO(&u)
	return r.CreateAndSync(ctx, po, func(updated *UserPO) {
		u.ID = user.UserID(updated.ID)
	})
}

// FindByID 根据ID查找用户
func (r *Repository) FindByID(ctx context.Context, id user.UserID) (user.User, error) {
	po, err := r.BaseRepository.FindByID(ctx, id.Value())
	if err != nil {
		return user.User{}, err
	}
	u := r.mapper.ToBO(po)
	if u == nil {
		return user.User{}, gorm.ErrRecordNotFound
	}
	return *u, nil
}

// FindByPhone 根据手机号查找用户
func (r *Repository) FindByPhone(ctx context.Context, phone meta.Phone) (user.User, error) {
	var po UserPO
	err := r.FindByField(ctx, &po, "phone", phone.String())
	if err != nil {
		return user.User{}, err
	}
	u := r.mapper.ToBO(&po)
	if u == nil {
		return user.User{}, gorm.ErrRecordNotFound
	}
	return *u, nil
}

// Update 更新用户信息
func (r *Repository) Update(ctx context.Context, u user.User) error {
	po := r.mapper.ToPO(&u)
	return r.UpdateAndSync(ctx, po, func(updated *UserPO) {
		u.ID = user.UserID(updated.ID)
	})
}
