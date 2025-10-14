package response

import (
	"time"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
)

// Account response DTO.
type Account struct {
	ID         string  `json:"id"`
	UserID     string  `json:"userId"`
	Provider   string  `json:"provider"`
	ExternalID string  `json:"externalId"`
	AppID      *string `json:"appId,omitempty"`
	Status     string  `json:"status"`
}

// OperationCredential response DTO (without hash).
type OperationCredential struct {
	AccountID      string     `json:"accountId"`
	Username       string     `json:"username"`
	LockedUntil    *time.Time `json:"lockedUntil,omitempty"`
	FailedAttempts int        `json:"failedAttempts"`
	LastChangedAt  time.Time  `json:"lastChangedAt"`
}

// OperationCredentialView bundles account + credential.
type OperationCredentialView struct {
	Account    Account             `json:"account"`
	Credential OperationCredential `json:"credential"`
}

// BindResult response.
type BindResult struct {
	AccountID string `json:"accountId"`
	Created   bool   `json:"created"`
}

// AccountPage paginated listing.
type AccountPage struct {
	Total  int       `json:"total"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
	Items  []Account `json:"items"`
}

// NewAccount from domain model.
func NewAccount(acc *domain.Account) Account {
	if acc == nil {
		return Account{}
	}
	resp := Account{
		ID:         idutil.ID(acc.ID).String(),
		UserID:     idutil.NewID(acc.UserID.Value()).String(),
		Provider:   string(acc.Provider),
		ExternalID: acc.ExternalID,
		Status:     statusToString(acc.Status),
	}
	if acc.AppID != nil {
		resp.AppID = acc.AppID
	}
	return resp
}

// NewOperationCredential from domain model.
func NewOperationCredential(cred *domain.OperationAccount) OperationCredential {
	if cred == nil {
		return OperationCredential{}
	}
	resp := OperationCredential{
		AccountID:      idutil.ID(cred.AccountID).String(),
		Username:       cred.Username,
		FailedAttempts: cred.FailedAttempts,
		LastChangedAt:  cred.LastChangedAt,
	}
	if cred.LockedUntil != nil {
		resp.LockedUntil = cred.LockedUntil
	}
	return resp
}

// NewOperationCredentialView builds combined response.
func NewOperationCredentialView(acc *domain.Account, cred *domain.OperationAccount) OperationCredentialView {
	return OperationCredentialView{
		Account:    NewAccount(acc),
		Credential: NewOperationCredential(cred),
	}
}

// NewBindResult builds bind result.
func NewBindResult(accountID domain.AccountID, created bool) BindResult {
	return BindResult{
		AccountID: idutil.ID(accountID).String(),
		Created:   created,
	}
}

// NewAccountPage builds paginated response.
func NewAccountPage(total, limit, offset int, items []*domain.Account) AccountPage {
	resp := AccountPage{
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Items:  make([]Account, 0, len(items)),
	}
	for _, item := range items {
		resp.Items = append(resp.Items, NewAccount(item))
	}
	return resp
}

func statusToString(status domain.AccountStatus) string {
	switch status {
	case domain.StatusDisabled:
		return "disabled"
	case domain.StatusActive:
		return "active"
	case domain.StatusArchived:
		return "archived"
	case domain.StatusDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}
