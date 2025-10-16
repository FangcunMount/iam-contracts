package service

import (
	"context"
	"errors"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	drivenPort "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/port/driven"
	drivingPort "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/port/driving"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
	"gorm.io/gorm"
)

// StatusService 负责账号状态的开启/禁用。
type StatusService struct {
	accounts drivenPort.AccountRepo
}

var _ drivingPort.AccountStatusUpdater = (*StatusService)(nil)

// NewStatusService 构造状态服务。
func NewStatusService(acc drivenPort.AccountRepo) *StatusService {
	return &StatusService{accounts: acc}
}

// DisableAccount 将账号标记为禁用。
func (s *StatusService) DisableAccount(ctx context.Context, accountID domain.AccountID) error {
	return s.updateStatus(ctx, accountID, domain.StatusDisabled)
}

// EnableAccount 将账号标记为启用。
func (s *StatusService) EnableAccount(ctx context.Context, accountID domain.AccountID) error {
	return s.updateStatus(ctx, accountID, domain.StatusActive)
}

func (s *StatusService) updateStatus(ctx context.Context, accountID domain.AccountID, status domain.AccountStatus) error {
	aid := accountIDString(accountID)
	if _, err := s.accounts.FindByID(ctx, accountID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return perrors.WithCode(code.ErrInvalidArgument, "account(%s) not found", aid)
		}
		return perrors.WrapC(err, code.ErrDatabase, "load account(%s) before status update failed", aid)
	}

	if err := s.accounts.UpdateStatus(ctx, accountID, status); err != nil {
		return perrors.WrapC(err, code.ErrDatabase, "update status for account(%s) failed", aid)
	}

	return nil
}
