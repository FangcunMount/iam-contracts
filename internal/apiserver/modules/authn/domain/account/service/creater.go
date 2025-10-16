package service

import (
	"strings"
	"time"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
)

const (
	defaultOperationAccountAlgo   = "plain"
	defaultOperationPasswordAddon = "123456"
)

// CreateAccount 构造基础账号实体，负责清洗输入并补齐默认状态。
func CreateAccount(
	userID domain.UserID,
	provider domain.Provider,
	externalID string,
	appID *string,
	opts ...domain.AccountOption,
) (*domain.Account, error) {
	if userID.IsZero() {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "user id cannot be empty")
	}

	providerValue := strings.TrimSpace(string(provider))
	if providerValue == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "provider cannot be empty")
	}

	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "external id cannot be empty")
	}

	accountOpts := []domain.AccountOption{
		domain.WithExternalID(externalID),
		domain.WithStatus(domain.StatusActive),
	}

	if appID != nil {
		value := strings.TrimSpace(*appID)
		if value == "" {
			return nil, perrors.WithCode(code.ErrInvalidArgument, "app id cannot be empty")
		}
		accountOpts = append(accountOpts, domain.WithAppID(value))
	}

	if len(opts) > 0 {
		accountOpts = append(accountOpts, opts...)
	}

	acc := domain.NewAccount(userID, domain.Provider(providerValue), accountOpts...)
	return &acc, nil
}

// CreateOperationAccount 构造运营后台账号凭证实体。
func CreateOperationAccount(
	accountID domain.AccountID,
	username string,
	algo string,
	opts ...domain.OperationAccountOption,
) (*domain.OperationAccount, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "username cannot be empty")
	}

	algo = strings.TrimSpace(algo)
	if algo == "" {
		algo = defaultOperationAccountAlgo
	}

	opOpts := []domain.OperationAccountOption{
		domain.WithPasswordHash([]byte(username + defaultOperationPasswordAddon)),
		domain.WithLastChangedAt(time.Now()),
	}
	if len(opts) > 0 {
		opOpts = append(opOpts, opts...)
	}

	cred := domain.NewOperationAccount(accountID, username, algo, opOpts...)
	return &cred, nil
}

// CreateWeChatAccount 构造微信账号凭证实体。
func CreateWeChatAccount(
	accountID domain.AccountID,
	appID string,
	openID string,
	opts ...domain.WeChatAccountOption,
) (*domain.WeChatAccount, error) {
	appID = strings.TrimSpace(appID)
	openID = strings.TrimSpace(openID)
	if appID == "" || openID == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "wechat app_id and open_id cannot be empty")
	}

	wxOpts := opts
	wx := domain.NewWeChatAccount(accountID, appID, openID, wxOpts...)
	return &wx, nil
}

func accountIDString(id domain.AccountID) string {
	return idutil.ID(id).String()
}
