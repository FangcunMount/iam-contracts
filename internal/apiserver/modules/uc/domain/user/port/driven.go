package port

import (
	"context"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
)

// ==================== Driven Ports (被驱动端口) ====================
// 这些接口由基础设施层实现，供领域层调用

// UserRepository 用户存储接口 - 被驱动端口
// 由 infrastructure 层实现，领域服务通过此接口访问数据
type UserRepository interface {
	Create(ctx context.Context, user *user.User) error
	FindByID(ctx context.Context, id user.UserID) (*user.User, error)
	FindByPhone(ctx context.Context, phone meta.Phone) (*user.User, error)
	Update(ctx context.Context, user *user.User) error
}
