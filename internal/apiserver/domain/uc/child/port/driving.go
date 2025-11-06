package port

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/child"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ==================== Driving Ports (驱动端口) ====================
// 这些接口由领域层（领域服务）实现，供应用层调用
// 按照功能职责拆分，遵循接口隔离原则

// ChildRegister 儿童注册领域服务接口
// 负责儿童档案注册相关的领域逻辑
type ChildRegister interface {
	Register(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) (*child.Child, error)
	RegisterWithIDCard(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday, idCard meta.IDCard) (*child.Child, error)
}

// ChildProfileEditor 儿童资料管理领域服务接口
// 负责儿童档案编辑相关的领域逻辑
type ChildProfileEditor interface {
	Rename(ctx context.Context, childID child.ChildID, name string) (*child.Child, error)
	UpdateIDCard(ctx context.Context, childID child.ChildID, idCard meta.IDCard) (*child.Child, error)
	UpdateProfile(ctx context.Context, childID child.ChildID, gender meta.Gender, birthday meta.Birthday) (*child.Child, error)
	UpdateHeightWeight(ctx context.Context, childID child.ChildID, height meta.Height, weight meta.Weight) (*child.Child, error)
}

// ChildQueryer 儿童查询领域服务接口
// 负责儿童档案查询相关的领域逻辑
type ChildQueryer interface {
	FindByID(ctx context.Context, childID child.ChildID) (*child.Child, error)
	FindByIDCard(ctx context.Context, idCard meta.IDCard) (*child.Child, error)
	FindListByName(ctx context.Context, name string) ([]*child.Child, error)
	FindListByNameAndBirthday(ctx context.Context, name string, birthday meta.Birthday) ([]*child.Child, error)
	FindSimilar(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) ([]*child.Child, error)
}
