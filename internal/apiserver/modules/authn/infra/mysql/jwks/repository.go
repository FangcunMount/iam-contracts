package jwks

import (
	"context"
	"time"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/jwks"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/jwks/port/driven"
	"gorm.io/gorm"
)

// KeyRepository MySQL 实现
type KeyRepository struct {
	mapper *Mapper
	db     *gorm.DB
}

var _ driven.KeyRepository = (*KeyRepository)(nil)

// NewKeyRepository 创建 KeyRepository 实例
func NewKeyRepository(db *gorm.DB) driven.KeyRepository {
	return &KeyRepository{
		mapper: NewMapper(),
		db:     db,
	}
}

// Save 保存密钥（创建）
func (r *KeyRepository) Save(ctx context.Context, key *jwks.Key) error {
	po, err := r.mapper.ToKeyPO(key)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Create(po).Error
}

// Update 更新密钥
func (r *KeyRepository) Update(ctx context.Context, key *jwks.Key) error {
	po, err := r.mapper.ToKeyPO(key)
	if err != nil {
		return err
	}

	// 使用 kid 作为 WHERE 条件
	return r.db.WithContext(ctx).
		Model(&KeyPO{}).
		Where("kid = ?", key.Kid).
		Updates(po).Error
}

// Delete 删除密钥（物理删除）
func (r *KeyRepository) Delete(ctx context.Context, kid string) error {
	return r.db.WithContext(ctx).Where("kid = ?", kid).Delete(&KeyPO{}).Error
}

// FindByKid 根据 kid 查询密钥
func (r *KeyRepository) FindByKid(ctx context.Context, kid string) (*jwks.Key, error) {
	var po KeyPO
	err := r.db.WithContext(ctx).Where("kid = ?", kid).First(&po).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return r.mapper.ToKeyEntity(&po)
}

// FindByStatus 根据状态查询密钥列表
func (r *KeyRepository) FindByStatus(ctx context.Context, status jwks.KeyStatus) ([]*jwks.Key, error) {
	var pos []*KeyPO
	err := r.db.WithContext(ctx).
		Where("status = ?", int8(status)).
		Order("created_at DESC").
		Find(&pos).Error
	if err != nil {
		return nil, err
	}

	return r.mapper.ToKeyEntities(pos)
}

// FindPublishable 查询可发布的密钥（Active + Grace）
func (r *KeyRepository) FindPublishable(ctx context.Context) ([]*jwks.Key, error) {
	var pos []*KeyPO
	now := time.Now()

	err := r.db.WithContext(ctx).
		Where("status IN (?)", []int8{int8(jwks.KeyActive), int8(jwks.KeyGrace)}).
		Where("(not_before IS NULL OR not_before <= ?)", now).
		Where("(not_after IS NULL OR not_after > ?)", now).
		Order("kid ASC").
		Find(&pos).Error
	if err != nil {
		return nil, err
	}

	return r.mapper.ToKeyEntities(pos)
}

// FindExpired 查询已过期的密钥
func (r *KeyRepository) FindExpired(ctx context.Context) ([]*jwks.Key, error) {
	var pos []*KeyPO
	now := time.Now()

	err := r.db.WithContext(ctx).
		Where("not_after IS NOT NULL AND not_after <= ?", now).
		Order("not_after ASC").
		Find(&pos).Error
	if err != nil {
		return nil, err
	}

	return r.mapper.ToKeyEntities(pos)
}

// FindAll 查询所有密钥（分页）
func (r *KeyRepository) FindAll(ctx context.Context, offset, limit int) ([]*jwks.Key, int64, error) {
	var pos []*KeyPO
	var total int64

	// 查询总数
	if err := r.db.WithContext(ctx).Model(&KeyPO{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询数据
	query := r.db.WithContext(ctx).
		Order("created_at DESC").
		Offset(offset)

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&pos).Error
	if err != nil {
		return nil, 0, err
	}

	entities, err := r.mapper.ToKeyEntities(pos)
	if err != nil {
		return nil, 0, err
	}

	return entities, total, nil
}

// CountByStatus 统计指定状态的密钥数量
func (r *KeyRepository) CountByStatus(ctx context.Context, status jwks.KeyStatus) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&KeyPO{}).
		Where("status = ?", int8(status)).
		Count(&count).Error

	return count, err
}
