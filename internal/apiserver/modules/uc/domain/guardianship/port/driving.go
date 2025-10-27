package port

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	guardianship "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/guardianship"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ==================== Driving Ports (驱动端口) ====================
// 这些接口由领域层（领域服务）实现，供应用层调用
// 按照功能职责拆分，遵循接口隔离原则

// GuardianshipManager 监护关系管理领域服务接口
// 负责监护关系建立和撤销相关的领域逻辑
type GuardianshipManager interface {
	AddGuardian(ctx context.Context, userID user.UserID, childID child.ChildID, relation guardianship.Relation) (*guardianship.Guardianship, error)
	RemoveGuardian(ctx context.Context, userID user.UserID, childID child.ChildID) (*guardianship.Guardianship, error)
}

// GuardianshipQueryer 监护关系查询领域服务接口
// 负责监护关系查询相关的领域逻辑
type GuardianshipQueryer interface {
	FindByUserIDAndChildID(ctx context.Context, userID user.UserID, childID child.ChildID) (*guardianship.Guardianship, error)
	FindByUserIDAndChildName(ctx context.Context, userID user.UserID, childName string) ([]*guardianship.Guardianship, error)
	FindListByChildID(ctx context.Context, childID child.ChildID) ([]*guardianship.Guardianship, error)
	FindListByUserID(ctx context.Context, userID user.UserID) ([]*guardianship.Guardianship, error)
	IsGuardian(ctx context.Context, userID user.UserID, childID child.ChildID) (bool, error)
}

// GuardianshipRegister 监护关系注册领域服务接口
// 负责同时注册儿童和监护关系的复杂用例
type GuardianshipRegister interface {
	RegisterChildWithGuardian(ctx context.Context, params RegisterChildWithGuardianParams) (*guardianship.Guardianship, *child.Child, error)
}

// RegisterChildWithGuardianParams 同时注册儿童和监护关系的参数
type RegisterChildWithGuardianParams struct {
	Name     string
	Gender   meta.Gender
	Birthday meta.Birthday
	IDCard   meta.IDCard           // 可选
	Height   *meta.Height          // 可选
	Weight   *meta.Weight          // 可选
	UserID   user.UserID           // 监护人ID
	Relation guardianship.Relation // 监护关系
}
