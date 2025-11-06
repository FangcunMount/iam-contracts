package child

import (
	"context"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ================== Domain Service Interfaces (Driving Ports) ==================
// 这些接口由领域层（领域服务）实现，供应用层调用
// 按照功能职责拆分，遵循接口隔离原则

// Register 儿童注册领域服务接口
// 负责儿童档案注册相关的领域逻辑
type Register interface {
	Register(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday) (*Child, error)
	RegisterWithIDCard(ctx context.Context, name string, gender meta.Gender, birthday meta.Birthday, idCard meta.IDCard) (*Child, error)
}

// ProfileEditor 儿童资料管理领域服务接口
// 负责儿童档案编辑相关的领域逻辑
type ProfileEditor interface {
	Rename(ctx context.Context, childID meta.ID, name string) (*Child, error)
	UpdateIDCard(ctx context.Context, childID meta.ID, idCard meta.IDCard) (*Child, error)
	UpdateProfile(ctx context.Context, childID meta.ID, gender meta.Gender, birthday meta.Birthday) (*Child, error)
	UpdateHeightWeight(ctx context.Context, childID meta.ID, height meta.Height, weight meta.Weight) (*Child, error)
}
