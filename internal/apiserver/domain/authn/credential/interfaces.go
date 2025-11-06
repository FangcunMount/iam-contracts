package credential

import "time"

// ==================== Driving Ports (驱动端口) ====================
// 由领域层实现，应用层使用
// 遵循接口隔离原则，按职责细分

// Binder 凭据绑定器接口（Driving Port）
// 职责：将外部认证信息绑定到账号，创建凭据实体
type Binder interface {
	Bind(spec BindSpec) (*Credential, error)
}

// Usage 凭据使用记录接口（Driving Port）
// 职责：记录凭据使用情况，管理失败计数和锁定
type Usage interface {
	EnsureUsable(c *Credential, now time.Time) error                           // 确保凭据可用（已启用且未锁定）
	RecordSuccess(c *Credential, now time.Time)                                // 记录认证成功，重置失败计数
	RecordFailure(c *Credential, now time.Time, p LockoutPolicy) (locked bool) // 记录认证失败，应用锁定策略
}

// Locker 凭据锁定管理接口（Driving Port）
// 职责：锁定/解锁凭据（行政动作，主要用于 password 类型）
type Locker interface {
	LockUntil(c *Credential, until time.Time) // 锁定凭据直到指定时间
	Unlock(c *Credential)                     // 解锁凭据
}

// Rotator 凭据材料轮换接口（Driving Port）
// 职责：轮换凭据材料（仅 password 的条件再哈希）
type Rotator interface {
	Rotate(c *Credential, newMaterial []byte, newAlgo *string) // 轮换凭据材料
}

// Lifecycle 凭据生命周期管理接口（Driving Port）
// 职责：启用/禁用凭据
type Lifecycle interface {
	Enable(c *Credential)  // 启用凭据
	Disable(c *Credential) // 禁用凭据
}
