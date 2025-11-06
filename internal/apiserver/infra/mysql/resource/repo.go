package resource

import (
	"context"
	"fmt"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/resource"
	drivenPort "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/resource/port/driven"
	"github.com/FangcunMount/iam-contracts/internal/pkg/database/mysql"
	"gorm.io/gorm"
)

// ResourceRepository Resource 仓储实现
type ResourceRepository struct {
	mysql.BaseRepository[*ResourcePO]
	mapper *Mapper
	db     *gorm.DB
}

var _ drivenPort.ResourceRepo = (*ResourceRepository)(nil)

// NewResourceRepository 创建 Resource 仓储
func NewResourceRepository(db *gorm.DB) drivenPort.ResourceRepo {
	return &ResourceRepository{
		BaseRepository: mysql.NewBaseRepository[*ResourcePO](db),
		mapper:         NewMapper(),
		db:             db,
	}
}

// Create 创建新资源
func (r *ResourceRepository) Create(ctx context.Context, res *resource.Resource) error {
	po := r.mapper.ToPO(res)

	return r.BaseRepository.CreateAndSync(ctx, po, func(updated *ResourcePO) {
		res.ID = resource.ResourceID(updated.ID)
	})
}

// Update 更新资源
func (r *ResourceRepository) Update(ctx context.Context, res *resource.Resource) error {
	po := r.mapper.ToPO(res)

	return r.BaseRepository.UpdateAndSync(ctx, po, func(updated *ResourcePO) {
		// Sync if needed
	})
}

// FindByID 根据ID查找资源
func (r *ResourceRepository) FindByID(ctx context.Context, id resource.ResourceID) (*resource.Resource, error) {
	po, err := r.BaseRepository.FindByID(ctx, id.Uint64())
	if err != nil {
		return nil, fmt.Errorf("failed to find resource: %w", err)
	}

	bo := r.mapper.ToBO(po)
	if bo == nil {
		return nil, gorm.ErrRecordNotFound
	}

	return bo, nil
}

// FindByKey 根据资源键查找
func (r *ResourceRepository) FindByKey(ctx context.Context, key string) (*resource.Resource, error) {
	var po ResourcePO

	// `key` is a reserved word in MySQL; quote the column name to avoid syntax errors
	err := r.db.WithContext(ctx).Where("`key` = ?", key).First(&po).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find resource by key: %w", err)
	}

	bo := r.mapper.ToBO(&po)
	return bo, nil
}

// ListByApp 列出应用的资源
func (r *ResourceRepository) ListByApp(ctx context.Context, appName string, offset, limit int) ([]*resource.Resource, int64, error) {
	var pos []*ResourcePO
	var total int64

	// 统计总数
	if err := r.db.WithContext(ctx).Model(&ResourcePO{}).Where("app_name = ?", appName).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count resources: %w", err)
	}

	// 查询列表
	err := r.db.WithContext(ctx).
		Where("app_name = ?", appName).
		Order("`key` ASC").
		Offset(offset).
		Limit(limit).
		Find(&pos).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list resources: %w", err)
	}

	bos := r.mapper.ToBOList(pos)

	return bos, total, nil
}

// ListByDomain 列出业务域的资源
func (r *ResourceRepository) ListByDomain(ctx context.Context, domain string, offset, limit int) ([]*resource.Resource, int64, error) {
	var pos []*ResourcePO
	var total int64

	// 统计总数
	if err := r.db.WithContext(ctx).Model(&ResourcePO{}).Where("domain = ?", domain).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count resources: %w", err)
	}

	// 查询列表
	err := r.db.WithContext(ctx).
		Where("domain = ?", domain).
		Order("`key` ASC").
		Offset(offset).
		Limit(limit).
		Find(&pos).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list resources: %w", err)
	}

	bos := r.mapper.ToBOList(pos)

	return bos, total, nil
}

// List 列出所有资源
func (r *ResourceRepository) List(ctx context.Context, offset, limit int) ([]*resource.Resource, int64, error) {
	var pos []*ResourcePO
	var total int64

	// 统计总数
	if err := r.db.WithContext(ctx).Model(&ResourcePO{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count resources: %w", err)
	}

	// 查询列表
	err := r.db.WithContext(ctx).
		Order("`key` ASC").
		Offset(offset).
		Limit(limit).
		Find(&pos).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list resources: %w", err)
	}

	bos := r.mapper.ToBOList(pos)

	return bos, total, nil
}

// ValidateAction 校验动作是否在资源的允许列表中
func (r *ResourceRepository) ValidateAction(ctx context.Context, resourceKey, action string) (bool, error) {
	res, err := r.FindByKey(ctx, resourceKey)
	if err != nil {
		return false, fmt.Errorf("failed to find resource: %w", err)
	}

	if res == nil {
		return false, fmt.Errorf("resource not found: %s", resourceKey)
	}

	return res.HasAction(action), nil
}

// Delete 删除资源（软删除）
func (r *ResourceRepository) Delete(ctx context.Context, id resource.ResourceID) error {
	err := r.BaseRepository.DeleteByID(ctx, id.Uint64())
	if err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	return nil
}
