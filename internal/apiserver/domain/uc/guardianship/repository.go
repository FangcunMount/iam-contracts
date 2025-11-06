package guardianship

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/child"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user"
)

// ================== Repository Interface (Driven Port) ==================
// 定义领域模型所依赖的仓储接口，由基础设施层提供实现

// Repository 监护关系存储接口
type Repository interface {
	Create(ctx context.Context, guardianship *Guardianship) error
	FindByID(ctx context.Context, id idutil.ID) (*Guardianship, error)
	FindByChildID(ctx context.Context, id child.ChildID) (guardianships []*Guardianship, err error)
	FindByUserID(ctx context.Context, id user.UserID) (guardianships []*Guardianship, err error)
	FindByUserIDAndChildID(ctx context.Context, userID user.UserID, childID child.ChildID) (*Guardianship, error)
	IsGuardian(ctx context.Context, userID user.UserID, childID child.ChildID) (bool, error)
	Update(ctx context.Context, guardianship *Guardianship) error
}
