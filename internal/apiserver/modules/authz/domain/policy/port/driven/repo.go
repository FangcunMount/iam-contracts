// Package driven 策略领域被驱动端口定义
package driven

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authz/domain/policy"
)

// PolicyVersionRepo 策略版本仓储接口
type PolicyVersionRepo interface {
	// GetOrCreate 获取或创建租户的策略版本
	GetOrCreate(ctx context.Context, tenantID string) (*domain.PolicyVersion, error)
	// Increment 递增版本号并记录变更
	Increment(ctx context.Context, tenantID, changedBy, reason string) (*domain.PolicyVersion, error)
	// GetCurrent 获取当前版本
	GetCurrent(ctx context.Context, tenantID string) (*domain.PolicyVersion, error)
}
