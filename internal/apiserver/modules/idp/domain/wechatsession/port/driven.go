package port

import (
	"context"
	"time"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatsession"
)

// ==================== Driven Ports (驱动端口) ====================
// 定义领域模型、领域服务所依赖的外部服务接口
// 基础设施适配层需提供实现

// WechatSessionRepository 微信会话存储库接口
type WechatSessionRepository interface {
	Get(ctx context.Context, appID, openID string) (*domain.WechatSession, error)
	Set(ctx context.Context, s *domain.WechatSession, ttl time.Duration) error
}
