package guardianship

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/guardianship"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/guardianship"
	"github.com/FangcunMount/iam-contracts/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"gorm.io/gorm"
)

// Repository 监护关系存储库实现
type Repository struct {
	mysql.BaseRepository[*GuardianshipPO]
	mapper *GuardianshipMapper
}

// NewRepository 创建监护关系存储库
func NewRepository(db *gorm.DB) guardianship.Repository {
	return &Repository{
		BaseRepository: mysql.NewBaseRepository[*GuardianshipPO](db),
		mapper:         NewGuardianshipMapper(),
	}
}

// Create 创建新的监护关系
func (r *Repository) Create(ctx context.Context, g *domain.Guardianship) error {
	po := r.mapper.ToPO(g)
	return r.CreateAndSync(ctx, po, func(updated *GuardianshipPO) {
		g.ID = updated.ID
		if updated.EstablishedAt.IsZero() {
			return
		}
		g.EstablishedAt = updated.EstablishedAt
	})
}

// FindByID 根据 ID 查找监护关系
func (r *Repository) FindByID(ctx context.Context, id idutil.ID) (*domain.Guardianship, error) {
	po, err := r.BaseRepository.FindByID(ctx, id.Uint64())
	if err != nil {
		return nil, err
	}
	g := r.mapper.ToBO(po)
	if g == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return g, nil
}

// FindByChildID 根据儿童 ID 查找监护关系
func (r *Repository) FindByChildID(ctx context.Context, id meta.ID) ([]*domain.Guardianship, error) {
	var pos []*GuardianshipPO
	if err := r.WithContext(ctx).Where("child_id = ?", id.ToUint64()).Find(&pos).Error; err != nil {
		return nil, err
	}

	return r.toDomainSlice(pos), nil
}

// FindByUserID 根据监护人 ID 查找监护关系
func (r *Repository) FindByUserID(ctx context.Context, id meta.ID) ([]*domain.Guardianship, error) {
	var pos []*GuardianshipPO
	if err := r.WithContext(ctx).Where("user_id = ?", id.ToUint64()).Find(&pos).Error; err != nil {
		return nil, err
	}

	return r.toDomainSlice(pos), nil
}

// FindByUserIDAndChildID 根据监护人 ID 和儿童 ID 查找监护关系
func (r *Repository) FindByUserIDAndChildID(ctx context.Context, userID meta.ID, childID meta.ID) (*domain.Guardianship, error) {
	var po GuardianshipPO
	if err := r.WithContext(ctx).Where("user_id = ? AND child_id = ?", userID.ToUint64(), childID.ToUint64()).First(&po).Error; err != nil {
		return nil, err
	}

	g := r.mapper.ToBO(&po)
	if g == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return g, nil
}

// IsGuardian 检查是否为监护关系
func (r *Repository) IsGuardian(ctx context.Context, userID meta.ID, childID meta.ID) (bool, error) {
	var count int64
	if err := r.WithContext(ctx).Model(&GuardianshipPO{}).
		Where("user_id = ? AND child_id = ?", userID.ToUint64(), childID.ToUint64()).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// Update 更新监护关系// Update 更新监护关系
func (r *Repository) Update(ctx context.Context, g *domain.Guardianship) error {
	po := r.mapper.ToPO(g)
	return r.UpdateAndSync(ctx, po, func(updated *GuardianshipPO) {
		g.ID = updated.ID
		g.EstablishedAt = updated.EstablishedAt
		g.RevokedAt = updated.RevokedAt
	})
}

func (r *Repository) toDomainSlice(pos []*GuardianshipPO) []*domain.Guardianship {
	bos := r.mapper.ToBOs(pos)
	guardianships := make([]*domain.Guardianship, 0, len(bos))
	for _, bo := range bos {
		if bo == nil {
			continue
		}
		guardianships = append(guardianships, bo)
	}
	return guardianships
}
