package service

import (
	"context"
	"errors"
	"strings"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	drivenPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port/driven"
	drivingPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port/driving"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	perrors "github.com/FangcunMount/iam-contracts/pkg/errors"
	"gorm.io/gorm"
)

// QueryService 提供账号查询能力。
type QueryService struct {
	accounts  drivenPort.AccountRepo
	wechat    drivenPort.WeChatRepo
	operation drivenPort.OperationRepo
}

var _ drivingPort.AccountQueryer = (*QueryService)(nil)

// NewQueryService 构造查询服务。
func NewQueryService(acc drivenPort.AccountRepo, wx drivenPort.WeChatRepo, op drivenPort.OperationRepo) *QueryService {
	return &QueryService{
		accounts:  acc,
		wechat:    wx,
		operation: op,
	}
}

// FindAccountByID 通过账号 ID 查询账号。
func (s *QueryService) FindAccountByID(ctx context.Context, accountID domain.AccountID) (*domain.Account, error) {
	aid := accountIDString(accountID)
	acc, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, perrors.WithCode(code.ErrInvalidArgument, "account(%s) not found", aid)
		}
		return nil, perrors.WrapC(err, code.ErrDatabase, "load account(%s) failed", aid)
	}

	return acc, nil
}

// FindByUsername 通过运营账号用户名查询账号及凭证。
func (s *QueryService) FindByUsername(ctx context.Context, username string) (*domain.Account, *domain.OperationAccount, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, nil, perrors.WithCode(code.ErrInvalidArgument, "username cannot be empty")
	}

	cred, err := s.operation.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, perrors.WithCode(code.ErrInvalidArgument, "operation credential(%s) not found", username)
		}
		return nil, nil, perrors.WrapC(err, code.ErrDatabase, "load operation credential(%s) failed", username)
	}

	credAccountID := cred.AccountID
	accIDStr := accountIDString(credAccountID)
	acc, err := s.accounts.FindByID(ctx, credAccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, perrors.WithCode(code.ErrInvalidArgument, "account(%s) not found", accIDStr)
		}
		return nil, nil, perrors.WrapC(err, code.ErrDatabase, "load account(%s) failed", accIDStr)
	}

	return acc, cred, nil
}

// FindByWeChatRef 根据微信 openid + appid 查询账号及凭证。
func (s *QueryService) FindByWeChatRef(ctx context.Context, externalID, appID string) (*domain.Account, *domain.WeChatAccount, error) {
	externalID = strings.TrimSpace(externalID)
	appID = strings.TrimSpace(appID)
	if externalID == "" || appID == "" {
		return nil, nil, perrors.WithCode(code.ErrInvalidArgument, "wechat external_id and app_id cannot be empty")
	}

	wx, err := s.wechat.FindByAppOpenID(ctx, appID, externalID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, perrors.WithCode(code.ErrInvalidArgument, "wechat credential not found")
		}
		return nil, nil, perrors.WrapC(err, code.ErrDatabase, "load wechat credential failed")
	}

	accIDStr := accountIDString(wx.AccountID)
	acc, err := s.accounts.FindByID(ctx, wx.AccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, perrors.WithCode(code.ErrInvalidArgument, "account(%s) not found", accIDStr)
		}
		return nil, nil, perrors.WrapC(err, code.ErrDatabase, "load account(%s) failed", accIDStr)
	}

	return acc, wx, nil
}

// FindByRef 按 provider/externalId/appId 查询账号。
func (s *QueryService) FindByRef(ctx context.Context, provider domain.Provider, externalID string, appID *string) (*domain.Account, error) {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "externalId cannot be empty")
	}

	var app *string
	if appID != nil {
		value := strings.TrimSpace(*appID)
		if value != "" {
			app = &value
		}
	}

	acc, err := s.accounts.FindByRef(ctx, provider, externalID, app)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, perrors.WithCode(code.ErrInvalidArgument, "account not found")
		}
		return nil, perrors.WrapC(err, code.ErrDatabase, "find account by ref failed")
	}
	return acc, nil
}

// FindAccountListByUserID 根据 UserID 查询账号列表。
func (s *QueryService) FindAccountListByUserID(ctx context.Context, userID domain.UserID) ([]*domain.Account, error) {
	type accountLister interface {
		ListByUserID(ctx context.Context, userID domain.UserID) ([]*domain.Account, error)
	}

	if lister, ok := s.accounts.(accountLister); ok {
		accounts, err := lister.ListByUserID(ctx, userID)
		if err != nil {
			return nil, perrors.WrapC(err, code.ErrDatabase, "load accounts by user(%s) failed", userID.String())
		}
		return accounts, nil
	}

	return nil, perrors.WithCode(code.ErrInternalServerError, "account repository does not support list by user id")
}
