package driving

import (
	"context"
	"time"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/jwks"
)

// KeyRotationService 密钥轮换服务接口
// 负责密钥的自动轮换：生成新密钥、旧密钥进入宽限期、清理过期密钥
// 由应用层调用（定时任务或手动触发），实现在领域服务层
type KeyRotationService interface {
	// RotateKey 执行密钥轮换
	// 轮换流程：
	// 1. 生成新密钥（Active 状态）
	// 2. 将当前 Active 密钥转为 Grace 状态
	// 3. 清理超过 MaxKeys 的密钥（将最老的 Grace 密钥转为 Retired）
	// 4. 清理过期的 Retired 密钥
	// 返回：新生成的密钥
	RotateKey(ctx context.Context) (*jwks.Key, error)

	// ShouldRotate 判断是否需要轮换
	// 根据 RotationPolicy 判断当前 Active 密钥是否已到轮换时间
	// 返回：true 表示需要轮换
	ShouldRotate(ctx context.Context) (bool, error)

	// GetRotationPolicy 获取当前轮换策略
	GetRotationPolicy() jwks.RotationPolicy

	// UpdateRotationPolicy 更新轮换策略
	UpdateRotationPolicy(ctx context.Context, policy jwks.RotationPolicy) error
}

// RotationStatus 轮换状态
type RotationStatus struct {
	// LastRotation 上次轮换时间
	LastRotation time.Time

	// NextRotation 下次计划轮换时间
	NextRotation time.Time

	// ActiveKey 当前激活的密钥信息
	ActiveKey *KeyInfo

	// GraceKeys 宽限期密钥列表
	GraceKeys []*KeyInfo

	// RetiredKeys 已退役密钥数量
	RetiredKeys int

	// Policy 当前轮换策略
	Policy jwks.RotationPolicy
}

// KeyInfo 密钥信息摘要
type KeyInfo struct {
	Kid       string
	Status    jwks.KeyStatus
	Algorithm string
	NotBefore *time.Time
	NotAfter  *time.Time
	CreatedAt time.Time
}
