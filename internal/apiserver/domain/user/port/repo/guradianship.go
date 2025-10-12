package repo

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
)

// GuardianshipRepository 监护关系存储接口
type GuardianshipRepository interface {
	Create(ctx context.Context, guardianship user.Guardianship) error
	FindByID(ctx context.Context, id idutil.ID) (user.Guardianship, error)
	FindByChildID(ctx context.Context, id user.ChildID) (guardianships []user.Guardianship, err error)
	FindByUserID(ctx context.Context, id user.UserID) (guardianships []user.Guardianship, err error)
	Update(ctx context.Context, guardianship user.Guardianship) error
}
