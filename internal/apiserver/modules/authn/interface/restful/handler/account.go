package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	domain "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
	req "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/interface/restful/request"
	resp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/interface/restful/response"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// AccountHandler exposes RESTful endpoints for account management.
type AccountHandler struct {
	*BaseHandler
	register port.AccountRegisterer
	editor   port.AccountEditor
	status   port.AccountStatusUpdater
	query    port.AccountQueryer
}

// NewAccountHandler constructs a new handler instance.
func NewAccountHandler(
	register port.AccountRegisterer,
	editor port.AccountEditor,
	status port.AccountStatusUpdater,
	query port.AccountQueryer,
) *AccountHandler {
	return &AccountHandler{
		BaseHandler: NewBaseHandler(),
		register:    register,
		editor:      editor,
		status:      status,
		query:       query,
	}
}

// CreateOperationAccount handles POST /v1/accounts/operation.
func (h *AccountHandler) CreateOperationAccount(c *gin.Context) {
	var reqBody req.CreateOperationAccountReq
	if err := h.BindJSON(c, &reqBody); err != nil {
		h.Error(c, err)
		return
	}

	if err := reqBody.Validate(); err != nil {
		h.Error(c, err)
		return
	}

	userID, err := parseUserID(reqBody.UserID)
	if err != nil {
		h.Error(c, err)
		return
	}

	username := strings.TrimSpace(reqBody.Username)
	account, _, err := h.register.CreateOperationAccount(c.Request.Context(), userID, username)
	if err != nil {
		h.Error(c, err)
		return
	}

	hash, algo, params, err := reqBody.HashPayload()
	if err != nil {
		h.Error(c, err)
		return
	}
	if hash != nil {
		if err := h.editor.UpdateOperationCredential(c.Request.Context(), username, hash, algo, params); err != nil {
			h.Error(c, err)
			return
		}
	}

	h.Created(c, resp.NewAccount(account))
}

// UpdateOperationCredential handles PATCH /v1/accounts/operation/{username}.
func (h *AccountHandler) UpdateOperationCredential(c *gin.Context) {
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		h.ErrorWithCode(c, code.ErrInvalidArgument, "username cannot be empty")
		return
	}

	var reqBody req.UpdateOperationCredentialReq
	if err := h.BindJSON(c, &reqBody); err != nil {
		h.Error(c, err)
		return
	}
	if err := reqBody.Validate(); err != nil {
		h.Error(c, err)
		return
	}

	if reqBody.NewPassword != nil || reqBody.NewHash != nil {
		hash, algo, params, err := reqBody.HashPayload()
		if err != nil {
			h.Error(c, err)
			return
		}
		if err := h.editor.UpdateOperationCredential(c.Request.Context(), username, hash, algo, params); err != nil {
			h.Error(c, err)
			return
		}
	}

	if reqBody.ResetFailures {
		if err := h.editor.ResetOperationFailures(c.Request.Context(), username); err != nil {
			h.Error(c, err)
			return
		}
	}

	if reqBody.UnlockNow {
		if err := h.editor.UnlockOperationAccount(c.Request.Context(), username); err != nil {
			h.Error(c, err)
			return
		}
	}

	h.Success(c, gin.H{"status": "ok"})
}

// ChangeOperationUsername handles POST /v1/accounts/operation/{username}:change.
func (h *AccountHandler) ChangeOperationUsername(c *gin.Context) {
	oldUsername := strings.TrimSpace(c.Param("username"))
	if oldUsername == "" {
		h.ErrorWithCode(c, code.ErrInvalidArgument, "username cannot be empty")
		return
	}

	var reqBody req.ChangeOperationUsernameReq
	if err := h.BindJSON(c, &reqBody); err != nil {
		h.Error(c, err)
		return
	}
	if err := reqBody.Validate(); err != nil {
		h.Error(c, err)
		return
	}

	if err := h.editor.ChangeOperationUsername(c.Request.Context(), oldUsername, strings.TrimSpace(reqBody.NewUsername)); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, gin.H{"status": "ok"})
}

// BindWeChatAccount handles POST /v1/accounts/wechat:bind.
func (h *AccountHandler) BindWeChatAccount(c *gin.Context) {
	var reqBody req.BindWeChatAccountReq
	if err := h.BindJSON(c, &reqBody); err != nil {
		h.Error(c, err)
		return
	}
	if err := reqBody.Validate(); err != nil {
		h.Error(c, err)
		return
	}

	userID, err := parseUserID(reqBody.UserID)
	if err != nil {
		h.Error(c, err)
		return
	}

	appID := strings.TrimSpace(reqBody.AppID)
	openID := strings.TrimSpace(reqBody.OpenID)

	ctx := c.Request.Context()

	existingAccount, _, err := h.query.FindByWeChatRef(ctx, openID, appID)
	created := false
	if err == nil {
		if existingAccount.UserID != userID {
			h.ErrorWithCode(c, code.ErrInvalidArgument, "wechat binding already associated with another user")
			return
		}
	} else {
		if !perrors.IsCode(err, code.ErrInvalidArgument) {
			h.Error(c, err)
			return
		}
		existingAccount = nil
	}

	var account *domain.Account
	if existingAccount == nil {
		account, _, err = h.register.CreateWeChatAccount(ctx, userID, openID, appID)
		if err != nil {
			h.Error(c, err)
			return
		}
		created = true
	} else {
		account = existingAccount
	}

	if err := h.upsertWeChatDetails(ctx, account.ID, &reqBody); err != nil {
		h.Error(c, err)
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, resp.NewBindResult(account.ID, created))
}

// UpsertWeChatProfile handles PATCH /v1/accounts/{accountId}/wechat:profile.
func (h *AccountHandler) UpsertWeChatProfile(c *gin.Context) {
	accountID, err := parseAccountID(c.Param("accountId"))
	if err != nil {
		h.Error(c, err)
		return
	}

	var reqBody req.UpsertWeChatProfileReq
	if err := h.BindJSON(c, &reqBody); err != nil {
		h.Error(c, err)
		return
	}
	if err := reqBody.Validate(); err != nil {
		h.Error(c, err)
		return
	}

	nickname, _ := normalizeOptionalString(reqBody.Nickname)
	avatar, _ := normalizeOptionalString(reqBody.Avatar)

	metaBytes, err := reqBody.MetaJSON()
	if err != nil {
		h.Error(c, err)
		return
	}

	if err := h.editor.UpdateWeChatProfile(c.Request.Context(), accountID, nickname, avatar, metaBytes); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, gin.H{"status": "ok"})
}

// SetWeChatUnionID handles PATCH /v1/accounts/{accountId}/wechat:unionid.
func (h *AccountHandler) SetWeChatUnionID(c *gin.Context) {
	accountID, err := parseAccountID(c.Param("accountId"))
	if err != nil {
		h.Error(c, err)
		return
	}

	var reqBody req.SetWeChatUnionIDReq
	if err := h.BindJSON(c, &reqBody); err != nil {
		h.Error(c, err)
		return
	}
	if err := reqBody.Validate(); err != nil {
		h.Error(c, err)
		return
	}

	if err := h.editor.SetWeChatUnionID(c.Request.Context(), accountID, strings.TrimSpace(reqBody.UnionID)); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, gin.H{"status": "ok"})
}

// GetAccount handles GET /v1/accounts/{accountId}.
func (h *AccountHandler) GetAccount(c *gin.Context) {
	accountID, err := parseAccountID(c.Param("accountId"))
	if err != nil {
		h.Error(c, err)
		return
	}

	account, err := h.query.FindAccountByID(c.Request.Context(), accountID)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, resp.NewAccount(account))
}

// EnableAccount handles POST /v1/accounts/{accountId}:enable.
func (h *AccountHandler) EnableAccount(c *gin.Context) {
	accountID, err := parseAccountID(c.Param("accountId"))
	if err != nil {
		h.Error(c, err)
		return
	}

	if err := h.status.EnableAccount(c.Request.Context(), accountID); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, gin.H{"status": "enabled"})
}

// DisableAccount handles POST /v1/accounts/{accountId}:disable.
func (h *AccountHandler) DisableAccount(c *gin.Context) {
	accountID, err := parseAccountID(c.Param("accountId"))
	if err != nil {
		h.Error(c, err)
		return
	}

	if err := h.status.DisableAccount(c.Request.Context(), accountID); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, gin.H{"status": "disabled"})
}

// ListAccountsByUser handles GET /v1/users/{userId}/accounts.
func (h *AccountHandler) ListAccountsByUser(c *gin.Context) {
	userID, err := parseUserID(c.Param("userId"))
	if err != nil {
		h.Error(c, err)
		return
	}

	limit := h.getQueryInt(c, "limit", 20, 1, 100)
	offset := h.getQueryInt(c, "offset", 0, 0, 1_000_000)

	accounts, err := h.query.FindAccountListByUserID(c.Request.Context(), userID)
	if err != nil {
		h.Error(c, err)
		return
	}

	total := len(accounts)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	paged := accounts[offset:end]
	h.Success(c, resp.NewAccountPage(total, limit, offset, paged))
}

// FindAccountByRef handles GET /v1/accounts:by-ref.
func (h *AccountHandler) FindAccountByRef(c *gin.Context) {
	providerRaw := strings.TrimSpace(c.Query("provider"))
	if providerRaw == "" {
		h.ErrorWithCode(c, code.ErrInvalidArgument, "provider is required")
		return
	}

	var provider domain.Provider
	switch domain.Provider(providerRaw) {
	case domain.ProviderPassword, domain.ProviderWeChat:
		provider = domain.Provider(providerRaw)
	default:
		h.ErrorWithCode(c, code.ErrInvalidArgument, "unsupported provider")
		return
	}

	externalID := strings.TrimSpace(c.Query("externalId"))
	if externalID == "" {
		h.ErrorWithCode(c, code.ErrInvalidArgument, "externalId is required")
		return
	}

	var appID *string
	if value := strings.TrimSpace(c.Query("appId")); value != "" {
		appID = &value
	}
	if provider == domain.ProviderWeChat && (appID == nil || *appID == "") {
		h.ErrorWithCode(c, code.ErrInvalidArgument, "appId is required for wechat provider")
		return
	}

	account, err := h.query.FindByRef(c.Request.Context(), provider, externalID, appID)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, resp.NewAccount(account))
}

// GetOperationAccountByUsername returns account + credential view by username.
func (h *AccountHandler) GetOperationAccountByUsername(c *gin.Context) {
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		h.ErrorWithCode(c, code.ErrInvalidArgument, "username cannot be empty")
		return
	}

	account, credential, err := h.query.FindByUsername(c.Request.Context(), username)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, resp.NewOperationCredentialView(account, credential))
}

func (h *AccountHandler) upsertWeChatDetails(ctx context.Context, accountID domain.AccountID, req *req.BindWeChatAccountReq) error {
	nickname, _ := normalizeOptionalString(req.Nickname)
	avatar, _ := normalizeOptionalString(req.Avatar)

	metaBytes, err := req.MetaJSON()
	if err != nil {
		return err
	}

	if err := h.editor.UpdateWeChatProfile(ctx, accountID, nickname, avatar, metaBytes); err != nil {
		return err
	}

	if req.UnionID != nil && strings.TrimSpace(*req.UnionID) != "" {
		if err := h.editor.SetWeChatUnionID(ctx, accountID, strings.TrimSpace(*req.UnionID)); err != nil {
			return err
		}
	}
	return nil
}

func (h *AccountHandler) getQueryInt(c *gin.Context, key string, defaultValue, min, max int) int {
	value := c.Query(key)
	if value == "" {
		return defaultValue
	}

	val, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func parseUserID(raw string) (domain.UserID, error) {
	value, err := ParseUint(strings.TrimSpace(raw), "user id")
	if err != nil {
		return 0, err
	}
	return domain.NewUserID(value), nil
}

func parseAccountID(raw string) (domain.AccountID, error) {
	value, err := ParseUint(strings.TrimSpace(raw), "account id")
	if err != nil {
		return domain.AccountID{}, err
	}
	return domain.NewAccountID(value), nil
}

func normalizeOptionalString(input *string) (*string, bool) {
	if input == nil {
		return nil, false
	}
	trimmed := strings.TrimSpace(*input)
	if trimmed == "" {
		return nil, true
	}
	return &trimmed, true
}
