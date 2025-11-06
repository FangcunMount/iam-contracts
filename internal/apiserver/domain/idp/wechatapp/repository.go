package wechatapp

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
)

// ================== Repository Interface (Driven Port) ==================
// 定义领域模型所依赖的仓储接口，由基础设施层提供实现

// Repository 微信应用存储库接口
type Repository interface {
	// 创建接口
	Create(ctx context.Context, app *WechatApp) error

	// 查询接口
	GetByID(ctx context.Context, id idutil.ID) (*WechatApp, error)
	GetByAppID(ctx context.Context, appID string) (*WechatApp, error)

	// 更新接口
	Update(ctx context.Context, app *WechatApp) error
}
