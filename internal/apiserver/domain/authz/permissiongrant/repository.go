package permissiongrant

import (
	"context"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

type Repository interface {
	Create(ctx context.Context, grant *Grant) error
	AtomicRevoke(ctx context.Context, id meta.ID) (RevokeOutcome, error)
	FindByID(ctx context.Context, id meta.ID) (*Grant, error)
	ListByRole(ctx context.Context, roleID meta.ID) ([]*Grant, error)
	ListActiveByResource(ctx context.Context, resourceID resource.ResourceID) ([]*Grant, error)
	ListActive(ctx context.Context) ([]*Grant, error)
}
