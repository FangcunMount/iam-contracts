package roleinheritance

import (
	"context"
	"errors"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	rolerepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/role"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/roleinheritance"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	mysql.BaseRepository[*InheritancePO]
	mapper Mapper
}

var _ domain.Repository = (*Repository)(nil)

func NewRepository(db *gorm.DB) domain.Repository {
	base := mysql.NewBaseRepository[*InheritancePO](db)
	base.SetErrorTranslator(mysql.NewDuplicateToTranslator(func(error) error {
		return perrors.WithCode(code.ErrRoleInheritanceAlreadyExists, "role inheritance already exists")
	}))
	return &Repository{BaseRepository: base}
}

func (r *Repository) CreateChecked(ctx context.Context, inheritance *domain.Inheritance) error {
	return r.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if inheritance == nil {
			return perrors.WithCode(code.ErrInvalidArgument, "inheritance required")
		}
		var roleRows []rolerepo.RolePO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Table("authz_roles").Where("deleted_at IS NULL").Order("id ASC").Find(&roleRows).Error; err != nil {
			return err
		}
		nodes := make([]domain.RoleNode, 0, len(roleRows))
		for _, row := range roleRows {
			nodes = append(nodes, domain.RoleNode{ID: row.ID, ManagementProtection: role.ManagementProtection(row.ManagementProtection)})
		}
		var rows []*InheritancePO
		query := tx.Where("revoked_at IS NULL AND deleted_at IS NULL").Order("id ASC")
		if tx.Dialector != nil && tx.Dialector.Name() != "sqlite" {
			query = query.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := query.Find(&rows).Error; err != nil {
			return err
		}
		existing := make([]*domain.Inheritance, 0, len(rows))
		for _, row := range rows {
			edge, err := r.mapper.ToBO(row)
			if err != nil {
				return err
			}
			existing = append(existing, edge)
		}
		if err := domain.ValidateGraph(nodes, append(existing, inheritance)); err != nil {
			return err
		}
		po := r.mapper.ToPO(inheritance)
		if err := tx.Create(po).Error; err != nil {
			return mysql.NewDuplicateToTranslator(func(error) error {
				return perrors.WithCode(code.ErrRoleInheritanceAlreadyExists, "role inheritance already exists")
			})(err)
		}
		inheritance.ID = po.ID
		inheritance.GrantedAt = po.GrantedAt
		inheritance.Version = po.Version
		return nil
	})
}

func (r *Repository) AtomicRevoke(ctx context.Context, id meta.ID) (domain.RevokeOutcome, error) {
	now := time.Now()
	query := r.WithContext(ctx).Model(&InheritancePO{}).
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
	var po InheritancePO
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

func (r *Repository) FindByID(ctx context.Context, id meta.ID) (*domain.Inheritance, error) {
	po, err := r.BaseRepository.FindByID(ctx, id.Uint64())
	if err != nil {
		return nil, err
	}
	return r.mapper.ToBO(po)
}

func (r *Repository) ListActive(ctx context.Context) ([]*domain.Inheritance, error) {
	var rows []*InheritancePO
	if err := r.WithContext(ctx).
		Where("revoked_at IS NULL AND deleted_at IS NULL").
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.Inheritance, 0, len(rows))
	for _, row := range rows {
		inheritance, err := r.mapper.ToBO(row)
		if err != nil {
			return nil, err
		}
		result = append(result, inheritance)
	}
	return result, nil
}
