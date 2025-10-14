package account

import (
	"time"
)

// OperationCredential 运营后台账号凭证
type OperationCredential struct {
	AccountID      AccountID
	Username       string     // 用户名
	PasswordHash   []byte     // 密码哈希
	Algo           string     // 哈希算法： bcrypt | argon2id | scrypt
	Params         []byte     // 算法参数，JSON 编码
	FailedAttempts int        // 连续失败尝试次数
	LockedUntil    *time.Time // 锁定截止时间，nil 表示未锁定
	LastChangedAt  time.Time  // 上次密码修改时间
}

// NewOperationCredential 创建运营后台账号凭证
func NewOperationCredential(id AccountID, username, algo string, opts ...OperationCredentialOption) OperationCredential {
	oc := OperationCredential{
		AccountID: id,
		Username:  username,
		Algo:      algo,
	}

	for _, opt := range opts {
		opt(&oc)
	}
	return oc
}

// OperationCredentialOption 运营后台账号凭证选项
type OperationCredentialOption func(*OperationCredential)

func WithPasswordHash(hash []byte) OperationCredentialOption {
	return func(c *OperationCredential) { c.PasswordHash = hash }
}
func WithParams(params []byte) OperationCredentialOption {
	return func(c *OperationCredential) { c.Params = params }
}
func WithFailedAttempts(attempts int) OperationCredentialOption {
	return func(c *OperationCredential) { c.FailedAttempts = attempts }
}
func WithLockedUntil(t *time.Time) OperationCredentialOption {
	return func(c *OperationCredential) { c.LockedUntil = t }
}
func WithLastChangedAt(t time.Time) OperationCredentialOption {
	return func(c *OperationCredential) { c.LastChangedAt = t }
}

// IsLocked 检查账号是否被锁定
func (c OperationCredential) IsLocked() bool {
	if c.LockedUntil == nil {
		return false
	}
	return c.LockedUntil.After(time.Now())
}

// Lock 锁定账号，直到指定时间
func (c *OperationCredential) Lock(until time.Time) {
	c.LockedUntil = &until
}

// Unlock 解锁账号
func (c *OperationCredential) Unlock() {
	c.LockedUntil = nil
	c.FailedAttempts = 0
}

// IncrementFailedAttempts 增加失败尝试次数
func (c *OperationCredential) IncrementFailedAttempts() {
	c.FailedAttempts++
}

// ResetFailedAttempts 重置失败尝试次数
func (c *OperationCredential) ResetFailedAttempts() {
	c.FailedAttempts = 0
}

// ChangePassword 修改密码
func (c *OperationCredential) ChangePassword(hash []byte, changedAt time.Time) {
	c.PasswordHash = hash
	c.LastChangedAt = changedAt
	c.ResetFailedAttempts()
	c.Unlock()
}

// IsPasswordExpired 检查密码是否过期
func (c OperationCredential) IsPasswordExpired(expiryDuration time.Duration) bool {
	if expiryDuration <= 0 {
		return false
	}
	return time.Since(c.LastChangedAt) > expiryDuration
}
