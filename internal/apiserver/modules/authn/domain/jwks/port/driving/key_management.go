package driving

import (
	"context"
	"time"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/jwks"
)

// KeyManagementService 密钥管理服务接口
// 负责密钥的生命周期管理：创建、激活、宽限、退役、清理
// 由应用层调用，实现在领域服务层
type KeyManagementService interface {
	// CreateKey 创建新密钥
	// alg: 签名算法（RS256/ES256/EdDSA 等）
	// notBefore: 生效时间（可选，默认为当前时间）
	// notAfter: 过期时间（可选，根据 RotationPolicy 计算）
	// 返回：创建的 Key 实体
	CreateKey(ctx context.Context, alg string, notBefore, notAfter *time.Time) (*jwks.Key, error)

	// GetActiveKey 获取当前激活的密钥
	// 用于 Token 签名时获取当前可用的密钥
	// 返回：当前 Active 状态且未过期的密钥
	GetActiveKey(ctx context.Context) (*jwks.Key, error)

	// GetKeyByKid 根据 kid 获取密钥
	// 用于验证 Token 时根据 kid 查找公钥
	GetKeyByKid(ctx context.Context, kid string) (*jwks.Key, error)

	// RetireKey 退役密钥（Grace → Retired）
	// 只能对 Grace 状态的密钥执行
	// kid: 密钥 ID
	RetireKey(ctx context.Context, kid string) error

	// ForceRetireKey 强制退役密钥（任何状态 → Retired）
	// 用于紧急情况（密钥泄露等）
	// kid: 密钥 ID
	ForceRetireKey(ctx context.Context, kid string) error

	// EnterGracePeriod 进入宽限期（Active → Grace）
	// 只能对 Active 状态的密钥执行
	// kid: 密钥 ID
	EnterGracePeriod(ctx context.Context, kid string) error

	// CleanupExpiredKeys 清理过期密钥
	// 删除 NotAfter < now 且 Status = Retired 的密钥
	// 返回：清理的密钥数量
	CleanupExpiredKeys(ctx context.Context) (int, error)

	// ListKeys 列出密钥（分页）
	// status: 可选的状态过滤（"", "active", "grace", "retired"）
	// limit/offset: 分页参数
	ListKeys(ctx context.Context, status jwks.KeyStatus, limit, offset int) ([]*jwks.Key, int64, error)
}
