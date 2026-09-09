package roleinheritance

import (
	"context"

	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// RevokeOutcome 撤销结果
type RevokeOutcome string

const (
	RevokeOutcomeRevoked        RevokeOutcome = "revoked"         // 本次成功撤销
	RevokeOutcomeAlreadyRevoked RevokeOutcome = "already_revoked" // 调用前已经撤销，本次未变更
	RevokeOutcomeNotFound       RevokeOutcome = "not_found"       // 未找到
)

// AtomicRevoker 原子撤销器
type AtomicRevoker interface {
	AtomicRevoke(ctx context.Context, id meta.ID) (RevokeOutcome, error)
}

// AppliesVersionChange 仅在本次改变撤销状态时要求发布新的授权事实版本。
func (o RevokeOutcome) AppliesVersionChange() bool {
	return o == RevokeOutcomeRevoked
}

// IsSuccess 判断撤销请求是否成功
func (o RevokeOutcome) IsSuccess() bool {
	return o == RevokeOutcomeRevoked
}
