package account

import (
	"context"
	"errors"
	"strings"
	"time"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	userdomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
	"gorm.io/gorm"
)

// RegisterService 负责注册账号及其子实体。
type RegisterService struct {
	accounts  port.AccountRepo
	wechat    port.WeChatRepo
	operation port.OperationRepo
}

var _ port.AccountRegisterer = (*RegisterService)(nil)

// NewRegisterService 构造注册服务。
func NewRegisterService(acc port.AccountRepo, wx port.WeChatRepo, op port.OperationRepo) *RegisterService {
	return &RegisterService{
		accounts:  acc,
		wechat:    wx,
		operation: op,
	}
}

// CreateOperationAccount 为用户创建（或返回已有的）运营后台账号。
func (s *RegisterService) CreateOperationAccount(
	ctx context.Context,
	userID userdomain.UserID,
	externalID string,
) (*domain.Account, *domain.OperationAccount, error) {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return nil, nil, perrors.WithCode(code.ErrInvalidArgument, "operation external_id cannot be empty")
	}

	// 确保账号存在
	account, _, err := s.ensureAccount(ctx, userID, domain.ProviderPassword, externalID, nil)
	if err != nil {
		return nil, nil, err
	}

	// 确保运营后台凭证存在
	cred, err := s.operation.FindByUsername(ctx, externalID)
	switch {
	case err == nil: // 已存在，直接返回
		if cred.AccountID != account.ID {
			return nil, nil, perrors.WithCode(code.ErrInvalidArgument, "operation credential belongs to another account")
		}
	case errors.Is(err, gorm.ErrRecordNotFound): // 不存在，创建新的
		created := domain.NewOperationAccount(
			account.ID,
			externalID,
			"",
			domain.WithLastChangedAt(time.Now()),
		)
		if err := s.operation.Create(ctx, &created); err != nil {
			return nil, nil, perrors.WrapC(err, code.ErrDatabase, "create operation credential failed")
		}
		cred = &created
	default:
		return nil, nil, perrors.WrapC(err, code.ErrDatabase, "load operation credential failed")
	}

	return account, cred, nil
}

// CreateWeChatAccount 为用户创建（或返回已有的）微信账号。
func (s *RegisterService) CreateWeChatAccount(
	ctx context.Context,
	userID userdomain.UserID,
	externalID string,
	appID string,
) (*domain.Account, *domain.WeChatAccount, error) {
	externalID = strings.TrimSpace(externalID)
	appID = strings.TrimSpace(appID)
	if externalID == "" || appID == "" {
		return nil, nil, perrors.WithCode(code.ErrInvalidArgument, "wechat external_id and app_id cannot be empty")
	}

	// 确保 Account 账号存在
	account, _, err := s.ensureAccount(ctx, userID, domain.ProviderWeChat, externalID, &appID)
	if err != nil {
		return nil, nil, err
	}

	// 确保 WeChat 账号存在
	wx, err := s.wechat.FindByAccountID(ctx, account.ID)
	switch {
	case err == nil: // 已存在，检查是否冲突
		if wx.OpenID != externalID || wx.AppID != appID {
			return nil, nil, perrors.WithCode(code.ErrInvalidArgument, "wechat credential conflicts with existing binding")
		}
	case errors.Is(err, gorm.ErrRecordNotFound): // 不存在，创建新的
		created := domain.NewWeChatAccount(account.ID, appID, externalID)
		if err := s.wechat.Create(ctx, &created); err != nil {
			return nil, nil, perrors.WrapC(err, code.ErrDatabase, "create wechat credential failed")
		}
		wx = &created
	default:
		return nil, nil, perrors.WrapC(err, code.ErrDatabase, "load wechat credential failed")
	}

	return account, wx, nil
}

// ensureAccount 确保指定的账号存在，若不存在则创建一个新的。
func (s *RegisterService) ensureAccount(
	ctx context.Context,
	userID userdomain.UserID,
	provider domain.Provider,
	externalID string,
	appID *string,
) (*domain.Account, bool, error) {
	acc, err := s.accounts.FindByRef(ctx, provider, externalID, appID)
	switch {
	case err == nil:
		if acc.UserID != userID {
			if err := s.accounts.UpdateUserID(ctx, acc.ID, userID); err != nil {
				return nil, false, perrors.WrapC(err, code.ErrDatabase, "bind user(%s) to account(%s) failed", userID.String(), accountIDString(acc.ID))
			}
			acc.UserID = userID
		}
		return acc, false, nil
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, false, perrors.WrapC(err, code.ErrDatabase, "query account by ref failed")
	}

	opts := []domain.AccountOption{
		domain.WithExternalID(externalID),
		domain.WithStatus(domain.StatusActive),
	}
	if appID != nil {
		opts = append(opts, domain.WithAppID(*appID))
	}

	newAccount := domain.NewAccount(userID, provider, opts...)
	if err := s.accounts.Create(ctx, &newAccount); err != nil {
		return nil, false, perrors.WrapC(err, code.ErrDatabase, "create account failed")
	}
	return &newAccount, true, nil
}
