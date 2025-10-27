package port

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ==================== Driving Ports (驱动端口) ====================
// 这些接口由领域层（领域服务）实现，供应用层调用
// 按照功能职责拆分，遵循接口隔离原则

// UserRegister 用户注册领域服务接口
// 负责用户注册相关的领域逻辑
type UserRegister interface {
	Register(ctx context.Context, name string, phone meta.Phone) (*user.User, error)
}

// UserProfileEditor 用户资料管理领域服务接口
// 负责用户资料编辑相关的领域逻辑
// 返回修改后的实体，由应用层负责持久化
type UserProfileEditor interface {
	Rename(ctx context.Context, userID user.UserID, name string) (*user.User, error)
	UpdateContact(ctx context.Context, userID user.UserID, phone meta.Phone, email meta.Email) (*user.User, error)
	UpdateIDCard(ctx context.Context, userID user.UserID, idCard meta.IDCard) (*user.User, error)
}

// UserStatusChanger 用户状态管理领域服务接口
// 负责用户状态变更相关的领域逻辑
// 返回修改后的实体，由应用层负责持久化
type UserStatusChanger interface {
	Activate(ctx context.Context, userID user.UserID) (*user.User, error)
	Deactivate(ctx context.Context, userID user.UserID) (*user.User, error)
	Block(ctx context.Context, userID user.UserID) (*user.User, error)
}

// UserQueryer 用户查询领域服务接口
// 负责用户查询相关的领域逻辑
type UserQueryer interface {
	FindByID(ctx context.Context, userID user.UserID) (*user.User, error)
	FindByPhone(ctx context.Context, phone meta.Phone) (*user.User, error)
}
