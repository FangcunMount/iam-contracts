package policy

import (
	"context"
	"fmt"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/policy"
	drivenPort "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/policy/port/driven"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/database/mysql"
	"gorm.io/gorm"
)

// PolicyVersionRepository PolicyVersion 仓储实现
type PolicyVersionRepository struct {
	mysql.BaseRepository[*PolicyVersionPO]
	mapper *Mapper
	db     *gorm.DB
}

var _ drivenPort.PolicyVersionRepo = (*PolicyVersionRepository)(nil)

// NewPolicyVersionRepository 创建 PolicyVersion 仓储
func NewPolicyVersionRepository(db *gorm.DB) drivenPort.PolicyVersionRepo {
	return &PolicyVersionRepository{
		BaseRepository: mysql.NewBaseRepository[*PolicyVersionPO](db),
		mapper:         NewMapper(),
		db:             db,
	}
}

// Create 创建新版本
func (r *PolicyVersionRepository) Create(ctx context.Context, pv *policy.PolicyVersion) error {
	po := r.mapper.ToPO(pv)

	return r.BaseRepository.CreateAndSync(ctx, po, func(updated *PolicyVersionPO) {
		pv.ID = policy.PolicyVersionID(updated.ID)
	})
}

// FindByID 根据ID查找版本
func (r *PolicyVersionRepository) FindByID(ctx context.Context, id policy.PolicyVersionID) (*policy.PolicyVersion, error) {
	po, err := r.BaseRepository.FindByID(ctx, id.Uint64())
	if err != nil {
		return nil, fmt.Errorf("failed to find policy version: %w", err)
	}

	bo := r.mapper.ToBO(po)
	if bo == nil {
		return nil, gorm.ErrRecordNotFound
	}

	return bo, nil
}

// GetCurrent 获取租户当前版本
func (r *PolicyVersionRepository) GetCurrent(ctx context.Context, tenantID string) (*policy.PolicyVersion, error) {
	var po PolicyVersionPO

	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("policy_version DESC").
		First(&po).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 租户没有版本记录，返回 nil
		}
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	bo := r.mapper.ToBO(&po)
	return bo, nil
}

// GetOrCreate 获取或创建租户的策略版本
func (r *PolicyVersionRepository) GetOrCreate(ctx context.Context, tenantID string) (*policy.PolicyVersion, error) {
	// 先尝试获取当前版本
	current, err := r.GetCurrent(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	// 如果已存在，直接返回
	if current != nil {
		return current, nil
	}

	// 不存在则创建初始版本
	newVersion := policy.NewPolicyVersion(
		tenantID,
		1, // 初始版本号为 1
		policy.WithChangedBy("system"),
		policy.WithReason("初始化策略版本"),
	)

	if err := r.Create(ctx, &newVersion); err != nil {
		return nil, fmt.Errorf("failed to create initial version: %w", err)
	}

	return &newVersion, nil
}

// Increment 递增版本号并记录变更
func (r *PolicyVersionRepository) Increment(ctx context.Context, tenantID, changedBy, reason string) (*policy.PolicyVersion, error) {
	// 获取当前版本号
	currentVersion, err := r.GetVersionNumber(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	// 创建新版本
	newVersion := policy.NewPolicyVersion(
		tenantID,
		currentVersion+1,
		policy.WithChangedBy(changedBy),
		policy.WithReason(reason),
	)

	if err := r.Create(ctx, &newVersion); err != nil {
		return nil, fmt.Errorf("failed to create new version: %w", err)
	}

	return &newVersion, nil
}

// GetVersionNumber 获取租户当前版本号
func (r *PolicyVersionRepository) GetVersionNumber(ctx context.Context, tenantID string) (int64, error) {
	pv, err := r.GetCurrent(ctx, tenantID)
	if err != nil {
		return 0, err
	}

	if pv == nil {
		return 0, nil // 没有版本记录，返回 0
	}

	return pv.Version, nil
}

// ListByTenant 列出租户的版本历史
func (r *PolicyVersionRepository) ListByTenant(ctx context.Context, tenantID string, offset, limit int) ([]*policy.PolicyVersion, int64, error) {
	var pos []*PolicyVersionPO
	var total int64

	// 统计总数
	if err := r.db.WithContext(ctx).Model(&PolicyVersionPO{}).Where("tenant_id = ?", tenantID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count policy versions: %w", err)
	}

	// 查询列表
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("policy_version DESC").
		Offset(offset).
		Limit(limit).
		Find(&pos).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list policy versions: %w", err)
	}

	bos := r.mapper.ToBOList(pos)

	return bos, total, nil
}

// Delete 删除版本（软删除）
func (r *PolicyVersionRepository) Delete(ctx context.Context, id policy.PolicyVersionID) error {
	err := r.BaseRepository.DeleteByID(ctx, id.Uint64())
	if err != nil {
		return fmt.Errorf("failed to delete policy version: %w", err)
	}

	return nil
}
