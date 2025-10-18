package role

import (
	"context"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/role"
	drivenPort "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/role/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/database/mysql"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
	"gorm.io/gorm"
)

// RoleRepository MySQL 实现
type RoleRepository struct {
	mysql.BaseRepository[*RolePO]
	mapper *Mapper
	db     *gorm.DB
}

var _ drivenPort.RoleRepo = (*RoleRepository)(nil)

// NewRoleRepository 构造函数
func NewRoleRepository(db *gorm.DB) drivenPort.RoleRepo {
	return &RoleRepository{
		BaseRepository: mysql.NewBaseRepository[*RolePO](db),
		mapper:         NewMapper(),
		db:             db,
	}
}

// Create 创建新角色
func (r *RoleRepository) Create(ctx context.Context, role *domain.Role) error {
	po := r.mapper.ToRolePO(role)
	return r.BaseRepository.CreateAndSync(ctx, po, func(updated *RolePO) {
		role.ID = domain.RoleID(updated.ID)
	})
}

// Update 更新角色
func (r *RoleRepository) Update(ctx context.Context, role *domain.Role) error {
	po := r.mapper.ToRolePO(role)
	return r.BaseRepository.UpdateAndSync(ctx, po, func(updated *RolePO) {
		// Sync if needed
	})
}

// Delete 删除角色
func (r *RoleRepository) Delete(ctx context.Context, id domain.RoleID) error {
	return r.BaseRepository.DeleteByID(ctx, idutil.ID(id).Uint64())
}

// FindByID 根据ID获取角色
func (r *RoleRepository) FindByID(ctx context.Context, id domain.RoleID) (*domain.Role, error) {
	po, err := r.BaseRepository.FindByID(ctx, idutil.ID(id).Uint64())
	if err != nil {
		return nil, err
	}
	role := r.mapper.ToRoleBO(po)
	if role == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return role, nil
}

// FindByName 根据名称和租户获取角色
func (r *RoleRepository) FindByName(ctx context.Context, tenantID, name string) (*domain.Role, error) {
	var po RolePO
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND name = ?", tenantID, name).First(&po).Error
	if err != nil {
		return nil, err
	}
	role := r.mapper.ToRoleBO(&po)
	if role == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return role, nil
}

// List 列出角色
func (r *RoleRepository) List(ctx context.Context, tenantID string, offset, limit int) ([]*domain.Role, int64, error) {
	var pos []*RolePO
	var total int64

	query := r.db.WithContext(ctx).Model(&RolePO{}).Where("tenant_id = ?", tenantID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	if err := query.Offset(offset).Limit(limit).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	roles := make([]*domain.Role, 0, len(pos))
	for _, po := range pos {
		if role := r.mapper.ToRoleBO(po); role != nil {
			roles = append(roles, role)
		}
	}

	return roles, total, nil
}
