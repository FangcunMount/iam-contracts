package guardianship

import (
	"context"

	childDomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child"
	guardport "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/guardianship/port"
	userDomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// PaternityExaminer 实现
type PaternityExaminer struct {
	repo guardport.GuardianshipRepository
}

var _ guardport.PaternityExaminer = (*PaternityExaminer)(nil)

// NewExaminer 创建亲子鉴定服务
func NewExaminer(r guardport.GuardianshipRepository) *PaternityExaminer {
	return &PaternityExaminer{repo: r}
}

// IsGuardian 实现检查是否为监护人
func (p *PaternityExaminer) IsGuardian(ctx context.Context, childID childDomain.ChildID, userID userDomain.UserID) (bool, error) {
	guardians, err := p.repo.FindByChildID(ctx, childID)
	if err != nil {
		return false, perrors.WrapC(err, code.ErrDatabase, "find guardians failed")
	}
	for _, g := range guardians {
		if g == nil {
			continue
		}
		if g.User == userID && g.IsActive() {
			return true, nil
		}
	}
	return false, nil
}
