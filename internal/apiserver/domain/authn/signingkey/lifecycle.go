// Package signingkey owns non-sensitive signing-key lifecycle rules.
// Cryptographic key material, wire serialization and persistence remain adapters.
package signingkey

import (
	"time"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// Status 密钥状态，表示一个密钥的生命周期状态。
// 它不是公共密钥集字段。
type Status uint8

const (
	StatusActive  Status = iota + 1 // 密钥处于 Active 状态，表示密钥处于活动状态，可以签名和验证。
	StatusGrace                     // 密钥处于 Grace 状态，表示密钥处于过渡期，可以验证但不能签名。
	StatusRetired                   // 密钥处于 Retired 状态，表示密钥已退役，不能再使用。
)

func (s Status) String() string {
	switch s {
	case StatusActive:
		return "active"
	case StatusGrace:
		return "grace"
	case StatusRetired:
		return "retired"
	default:
		return "unknown"
	}
}

// CanSign 判断是否可以签名
func (s Status) CanSign() bool {
	return s == StatusActive
}

// CanVerify 判断是否可以验证
func (s Status) CanVerify() bool {
	return s == StatusActive || s == StatusGrace
}

// EnterGrace 进入 Grace 状态
func (s Status) EnterGrace() (Status, error) {
	if s != StatusActive {
		return s, errors.WithCode(code.ErrInvalidStateTransition, "can only enter grace period from active state")
	}
	return StatusGrace, nil
}

// Retire 退役一个密钥
func (s Status) Retire() (Status, error) {
	if s != StatusGrace {
		return s, errors.WithCode(code.ErrInvalidStateTransition, "can only retire from grace period")
	}
	return StatusRetired, nil
}

// RotationPolicy 旋转策略，约束自动密钥旋转和公共重叠。
type RotationPolicy struct {
	RotationInterval   time.Duration // 旋转间隔
	GracePeriod        time.Duration // 宽容期
	MaxPublishableKeys int           // 最大可发布密钥数
}

// DefaultRotationPolicy 默认旋转策略
// 默认旋转间隔为 30 天，宽容期为 7 天，最大可发布密钥数为 3。
func DefaultRotationPolicy() RotationPolicy {
	return RotationPolicy{
		RotationInterval:   30 * 24 * time.Hour,
		GracePeriod:        7 * 24 * time.Hour,
		MaxPublishableKeys: 3,
	}
}

// Validate 验证旋转策略
func (p RotationPolicy) Validate() error {
	if p.RotationInterval <= 0 {
		return errors.WithCode(code.ErrInvalidRotationInterval, "rotation interval must be positive")
	}
	if p.GracePeriod <= 0 {
		return errors.WithCode(code.ErrInvalidGracePeriod, "grace period must be positive")
	}
	if p.MaxPublishableKeys < 2 {
		return errors.WithCode(code.ErrInvalidMaxKeys, "max keys must be at least 2")
	}
	if p.GracePeriod >= p.RotationInterval {
		return errors.WithCode(code.ErrGracePeriodTooLong, "grace period must be shorter than rotation interval")
	}
	return nil
}
