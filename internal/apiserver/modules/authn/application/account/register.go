package account

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/adapter"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/uow"
	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
	"gorm.io/gorm"
)

// RegisterService 负责注册账号及其子实体。
type RegisterService struct {
	accounts    port.AccountRepo
	wechat      port.WeChatRepo
	operation   port.OperationRepo
	uow         uow.UnitOfWork
	userAdapter adapter.UserAdapter
}

var _ port.AccountRegisterer = (*RegisterService)(nil)

// NewRegisterService 构造注册服务。
func NewRegisterService(
	acc port.AccountRepo,
	wx port.WeChatRepo,
	op port.OperationRepo,
	u uow.UnitOfWork,
	ua adapter.UserAdapter,
) *RegisterService {
	return &RegisterService{
		accounts:    acc,
		wechat:      wx,
		operation:   op,
		uow:         u,
		userAdapter: ua,
	}
}

// CreateOperationAccount 为用户创建（或返回已有的）运营后台账号。
func (s *RegisterService) CreateOperationAccount(
	ctx context.Context,
	userID domain.UserID,
	externalID string,
) (*domain.Account, *domain.OperationAccount, error) {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return nil, nil, perrors.WithCode(code.ErrInvalidArgument, "operation external_id cannot be empty")
	}

	// 校验 UserID 对应的用户是否存在
	exists, err := s.userAdapter.ExistsUser(ctx, userID)
	if err != nil {
		return nil, nil, perrors.WrapC(err, code.ErrInternalServerError, "check user existence failed")
	}
	if !exists {
		return nil, nil, perrors.WithCode(code.ErrUserNotFound, "user(%s) not found", userID.String())
	}

	var account *domain.Account
	var cred *domain.OperationAccount
	if err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		accRepo := pickAccountRepo(tx, s.accounts)
		opRepo := pickOperationRepo(tx, s.operation)
		if accRepo == nil || opRepo == nil {
			return perrors.WithCode(code.ErrInternalServerError, "operation repositories not configured")
		}

		created := false
		var err error
		account, created, err = ensureAccountWithRepo(ctx, accRepo, userID, domain.ProviderPassword, externalID, nil)
		if err != nil {
			return err
		}

		existing, err := opRepo.FindByUsername(ctx, externalID)
		switch {
		case err == nil:
			if existing.AccountID != account.ID {
				return perrors.WithCode(code.ErrInvalidArgument, "operation credential belongs to another account")
			}
			cred = existing
			return nil
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return perrors.WrapC(err, code.ErrDatabase, "load operation credential failed")
		default:
			if !created {
				return perrors.WithCode(code.ErrInvalidArgument, "operation credential not found for existing account")
			}

			initialPassword := externalID + "123456"
			newCred := domain.NewOperationAccount(
				account.ID,
				externalID,
				"plain",
				domain.WithPasswordHash([]byte(initialPassword)),
				domain.WithLastChangedAt(time.Now()),
			)
			if err := opRepo.Create(ctx, &newCred); err != nil {
				return perrors.WrapC(err, code.ErrDatabase, "create operation credential failed")
			}
			cred = &newCred
			return nil
		}
	}); err != nil {
		return nil, nil, err
	}

	return account, cred, nil
}

// CreateWeChatAccount 为用户创建（或返回已有的）微信账号。
func (s *RegisterService) CreateWeChatAccount(
	ctx context.Context,
	userID domain.UserID,
	externalID string,
	appID string,
) (*domain.Account, *domain.WeChatAccount, error) {
	externalID = strings.TrimSpace(externalID)
	appID = strings.TrimSpace(appID)
	if externalID == "" || appID == "" {
		return nil, nil, perrors.WithCode(code.ErrInvalidArgument, "wechat external_id and app_id cannot be empty")
	}

	// 校验 UserID 对应的用户是否存在
	exists, err := s.userAdapter.ExistsUser(ctx, userID)
	if err != nil {
		return nil, nil, perrors.WrapC(err, code.ErrInternalServerError, "check user existence failed")
	}
	if !exists {
		return nil, nil, perrors.WithCode(code.ErrUserNotFound, "user(%s) not found", userID.String())
	}

	var account *domain.Account
	var wx *domain.WeChatAccount
	if err := s.uow.WithinTx(ctx, func(tx uow.TxRepositories) error {
		accRepo := pickAccountRepo(tx, s.accounts)
		wxRepo := pickWeChatRepo(tx, s.wechat)
		if accRepo == nil || wxRepo == nil {
			return perrors.WithCode(code.ErrInternalServerError, "wechat repositories not configured")
		}

		app := appID // create local copy for pointer usage
		created := false
		var err error
		account, created, err = ensureAccountWithRepo(ctx, accRepo, userID, domain.ProviderWeChat, externalID, &app)
		if err != nil {
			return err
		}

		existing, err := wxRepo.FindByAccountID(ctx, account.ID)
		switch {
		case err == nil:
			if existing.OpenID != externalID || existing.AppID != appID {
				return perrors.WithCode(code.ErrInvalidArgument, "wechat credential conflicts with existing binding")
			}
			wx = existing
			return nil
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return perrors.WrapC(err, code.ErrDatabase, "load wechat credential failed")
		default:
			if !created {
				return perrors.WithCode(code.ErrInvalidArgument, "wechat credential not found for existing account")
			}

			newWx := domain.NewWeChatAccount(account.ID, appID, externalID)
			if err := wxRepo.Create(ctx, &newWx); err != nil {
				return perrors.WrapC(err, code.ErrDatabase, "create wechat credential failed")
			}
			wx = &newWx
			return nil
		}
	}); err != nil {
		return nil, nil, err
	}

	return account, wx, nil
}

func ensureAccountWithRepo(
	ctx context.Context,
	repo port.AccountRepo,
	userID domain.UserID,
	provider domain.Provider,
	externalID string,
	appID *string,
) (*domain.Account, bool, error) {
	if repo == nil {
		return nil, false, perrors.WithCode(code.ErrInternalServerError, "account repository not configured")
	}

	acc, err := repo.FindByRef(ctx, provider, externalID, appID)
	switch {
	case err == nil:
		// 如果已有账号,检查是否需要更新 UserID
		if acc.UserID != userID {
			if err := repo.UpdateUserID(ctx, acc.ID, userID); err != nil {
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
	if err := repo.Create(ctx, &newAccount); err != nil {
		return nil, false, perrors.WrapC(err, code.ErrDatabase, "create account failed")
	}
	return &newAccount, true, nil
}

func pickAccountRepo(tx uow.TxRepositories, fallback port.AccountRepo) port.AccountRepo {
	if tx.Accounts != nil {
		return tx.Accounts
	}
	return fallback
}

func pickOperationRepo(tx uow.TxRepositories, fallback port.OperationRepo) port.OperationRepo {
	if tx.Operation != nil {
		return tx.Operation
	}
	return fallback
}

func pickWeChatRepo(tx uow.TxRepositories, fallback port.WeChatRepo) port.WeChatRepo {
	if tx.WeChats != nil {
		return tx.WeChats
	}
	return fallback
}
