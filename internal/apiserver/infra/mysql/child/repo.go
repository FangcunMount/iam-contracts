package child

import (
	"context"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child/port"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/infra/mysql"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
	"gorm.io/gorm"
)

// Repository 儿童档案存储库实现
type Repository struct {
	mysql.BaseRepository[*ChildPO]
	mapper *ChildMapper
}

// NewRepository 创建儿童档案存储库
func NewRepository(db *gorm.DB) port.ChildRepository {
	return &Repository{
		BaseRepository: mysql.NewBaseRepository[*ChildPO](db),
		mapper:         NewChildMapper(),
	}
}

// Create 创建新的儿童档案
func (r *Repository) Create(ctx context.Context, child domain.Child) error {
	po := r.mapper.ToPO(&child)
	return r.CreateAndSync(ctx, po, func(updated *ChildPO) {
		child.ID = domain.NewChildID(updated.ID.Value())
	})
}

// FindByID 根据 ID 查找儿童档案
func (r *Repository) FindByID(ctx context.Context, id domain.ChildID) (domain.Child, error) {
	po, err := r.BaseRepository.FindByID(ctx, id.Value())
	if err != nil {
		return domain.Child{}, err
	}
	c := r.mapper.ToBO(po)
	if c == nil {
		return domain.Child{}, gorm.ErrRecordNotFound
	}
	return *c, nil
}

// FindByName 根据姓名查找儿童档案
func (r *Repository) FindByName(ctx context.Context, name string) (domain.Child, error) {
	var po ChildPO
	err := r.FindByField(ctx, &po, "name", name)
	if err != nil {
		return domain.Child{}, err
	}
	c := r.mapper.ToBO(&po)
	if c == nil {
		return domain.Child{}, gorm.ErrRecordNotFound
	}
	return *c, nil
}

// FindSimilar 根据姓名 + 性别 + 出生日期查找相似档案
func (r *Repository) FindSimilar(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) ([]domain.Child, error) {
	var pos []*ChildPO

	db := r.WithContext(ctx)
	if name != "" {
		db = db.Where("name = ?", name)
	}
	if gender.Value() != 0 {
		db = db.Where("gender = ?", gender.Value())
	}
	if !birthday.IsEmpty() {
		db = db.Where("birthday = ?", birthday.String())
	}

	if err := db.Find(&pos).Error; err != nil {
		return nil, err
	}

	bos := r.mapper.ToBOs(pos)
	children := make([]domain.Child, 0, len(bos))
	for _, bo := range bos {
		if bo == nil {
			continue
		}
		children = append(children, *bo)
	}

	return children, nil
}

// Update 更新儿童档案信息
func (r *Repository) Update(ctx context.Context, child domain.Child) error {
	po := r.mapper.ToPO(&child)
	return r.UpdateAndSync(ctx, po, func(updated *ChildPO) {
		child.ID = domain.NewChildID(updated.ID.Value())
	})
}
