// Package policy 策略领域包
package policy

import (
	"context"
)

// Repository 策略版本仓储接口（Driven Port）
type Repository interface {
	// GetOrCreate 获取或创建租户的策略版本
	GetOrCreate(ctx context.Context, tenantID string) (*PolicyVersion, error)
	// Increment 递增版本号并记录变更
	Increment(ctx context.Context, tenantID, changedBy, reason string) (*PolicyVersion, error)
	// GetCurrent 获取当前版本
	GetCurrent(ctx context.Context, tenantID string) (*PolicyVersion, error)
}
