package port

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession"
)

// ==================== Driving Ports (驱动端口) ====================
// 这些接口由领域层（领域服务）实现，供应用层调用
// 按照功能职责拆分，遵循接口隔离原则

// 3) 认证器（微信登录、产出统一声明）
type Authenticator interface {
	LoginWithCode(ctx context.Context, appID string, jsCode string) (*domain.ExternalClaim, *domain.WechatSession, error)
	DecryptPhone(ctx context.Context, appID, openID, encryptedData, iv string) (string, error)
}
