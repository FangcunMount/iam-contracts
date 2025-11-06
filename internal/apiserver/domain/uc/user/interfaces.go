package user

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ================== Domain Service Interfaces (Driving Ports) ==================
// 这些接口由领域层（领域服务）实现，供应用层调用
// 按照功能职责拆分，遵循接口隔离原则

// Register 用户注册领域服务接口
// 负责用户注册相关的领域逻辑
type Register interface {
	Register(ctx context.Context, name string, phone meta.Phone) (*User, error)
}

// ProfileEditor 用户资料管理领域服务接口
// 负责用户资料编辑相关的领域逻辑
// 返回修改后的实体，由应用层负责持久化
type ProfileEditor interface {
	Rename(ctx context.Context, userID UserID, name string) (*User, error)
	UpdateContact(ctx context.Context, userID UserID, phone meta.Phone, email meta.Email) (*User, error)
	UpdateIDCard(ctx context.Context, userID UserID, idCard meta.IDCard) (*User, error)
}

// StatusChanger 用户状态管理领域服务接口
// 负责用户状态变更相关的领域逻辑
// 返回修改后的实体，由应用层负责持久化
type StatusChanger interface {
	Activate(ctx context.Context, userID UserID) (*User, error)
	Deactivate(ctx context.Context, userID UserID) (*User, error)
	Block(ctx context.Context, userID UserID) (*User, error)
}
