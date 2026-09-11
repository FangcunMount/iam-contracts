package permissiongrant

import (
	"context"
	"errors"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	mysql.BaseRepository[*GrantPO]
	mapper Mapper
}

var _ domain.Repository = (*Repository)(nil)

func NewRepository(db *gorm.DB) domain.Repository {
	base := mysql.NewBaseRepository[*GrantPO](db)
	base.SetErrorTranslator(mysql.NewDuplicateToTranslator(func(error) error {
		return perrors.WithCode(code.ErrPermissionGrantAlreadyExists, "permission grant already exists")
	}))
	return &Repository{BaseRepository: base}
}

func (r *Repository) Create(ctx context.Context, grant *domain.Grant) error {
	po, err := r.mapper.ToPO(grant)
	if err != nil {
		return err
	}
	return r.CreateAndSync(ctx, po, func(updated *GrantPO) {
		grant.ID = updated.ID
		grant.GrantedAt = updated.GrantedAt
		grant.Version = updated.Version
	})
}

func (r *Repository) AtomicRevoke(ctx context.Context, id meta.ID) (domain.RevokeOutcome, error) {
	now := time.Now()
	query := r.WithContext(ctx).Model(&GrantPO{}).
		Where("id = ? AND revoked_at IS NULL", id.Uint64())
	result := query.Updates(map[string]any{
		"revoked_at": now,
		"updated_at": now,
		"updated_by": mysql.UserIDOrZero(ctx).Uint64(),
		"version":    gorm.Expr("version + 1"),
	})
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 1 {
		return domain.RevokeOutcomeRevoked, nil
	}
	var po GrantPO
	findQuery := r.WithContext(ctx)
	if findQuery.Dialector != nil && findQuery.Dialector.Name() != "sqlite" {
		findQuery = findQuery.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	findQuery = findQuery.Where("id = ?", id.Uint64())
	if err := findQuery.First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.RevokeOutcomeNotFound, nil
		}
		return "", err
	}
	if po.RevokedAt != nil {
		return domain.RevokeOutcomeAlreadyRevoked, nil
	}
	return domain.RevokeOutcomeNotFound, nil
}

// FindRoleID deliberately bypasses domain restoration: revoked legacy conditions
// remain audit data, while role protection still applies before AtomicRevoke.
func (r *Repository) FindRoleID(ctx context.Context, id meta.ID) (meta.ID, error) {
	po, err := r.BaseRepository.FindByID(ctx, id.Uint64())
	if err != nil {
		return meta.ID(0), err
	}
	return meta.FromUint64(po.RoleID), nil
}

func (r *Repository) FindByID(ctx context.Context, id meta.ID) (*domain.Grant, error) {
	po, err := r.BaseRepository.FindByID(ctx, id.Uint64())
	if err != nil {
		return nil, err
	}
	return r.mapper.ToBO(po)
}

func (r *Repository) ListByRole(ctx context.Context, roleID meta.ID) ([]*domain.Grant, error) {
	return r.list(ctx, "role_id = ? AND revoked_at IS NULL", roleID.Uint64())
}

func (r *Repository) ListActive(ctx context.Context) ([]*domain.Grant, error) {
	return r.list(ctx, "revoked_at IS NULL")
}

func (r *Repository) ListActiveByResource(ctx context.Context, resourceID resource.ResourceID) ([]*domain.Grant, error) {
	return r.list(ctx, "resource_id = ? AND revoked_at IS NULL", resourceID.Uint64())
}

func (r *Repository) list(ctx context.Context, query string, args ...any) ([]*domain.Grant, error) {
	var rows []*GrantPO
	if err := r.WithContext(ctx).Where(query, args...).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	grants := make([]*domain.Grant, 0, len(rows))
	for _, row := range rows {
		grant, err := r.mapper.ToBO(row)
		if err != nil {
			return nil, err
		}
		grants = append(grants, grant)
	}
	return grants, nil
}
