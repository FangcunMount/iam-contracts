package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	appAccount "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/account"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	req "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/interface/restful/request"
	resp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/interface/restful/response"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	perrors "github.com/FangcunMount/iam-contracts/pkg/errors"
)

// AccountHandler exposes RESTful endpoints for account management.
type AccountHandler struct {
	*BaseHandler
	accountService          appAccount.AccountApplicationService
	operationAccountService appAccount.OperationAccountApplicationService
	wechatAccountService    appAccount.WeChatAccountApplicationService
	lookupService           appAccount.AccountLookupApplicationService
}

// NewAccountHandler constructs a new handler instance.
func NewAccountHandler(
	accountService appAccount.AccountApplicationService,
	operationAccountService appAccount.OperationAccountApplicationService,
	wechatAccountService appAccount.WeChatAccountApplicationService,
	lookupService appAccount.AccountLookupApplicationService,
) *AccountHandler {
	return &AccountHandler{
		BaseHandler:             NewBaseHandler(),
		accountService:          accountService,
		operationAccountService: operationAccountService,
		wechatAccountService:    wechatAccountService,
		lookupService:           lookupService,
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

	// 获取密码哈希
	hash, algo, _, err := reqBody.HashPayload()
	if err != nil {
		h.Error(c, err)
		return
	}

	// 构建 DTO
	dto := appAccount.CreateOperationAccountDTO{
		UserID:   userID,
		Username: strings.TrimSpace(reqBody.Username),
		Password: "", // 如果有hash则在下面设置
		HashAlgo: algo,
	}
	if hash != nil {
		dto.Password = string(hash) // TODO: 这里应该使用密码适配器
	}

	// 调用应用服务
	result, err := h.accountService.CreateOperationAccount(c.Request.Context(), dto)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Created(c, resp.NewAccount(result.Account))
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

	// 更新凭据
	if reqBody.NewPassword != nil || reqBody.NewHash != nil {
		hash, algo, _, err := reqBody.HashPayload()
		if err != nil {
			h.Error(c, err)
			return
		}

		dto := appAccount.UpdateOperationCredentialDTO{
			Username: username,
			Password: string(hash), // TODO: 应该使用密码适配器
			HashAlgo: algo,
		}

		if err := h.operationAccountService.UpdateCredential(c.Request.Context(), dto); err != nil {
			h.Error(c, err)
			return
		}
	}

	// 重置失败次数
	if reqBody.ResetFailures {
		if err := h.operationAccountService.ResetFailures(c.Request.Context(), username); err != nil {
			h.Error(c, err)
			return
		}
	}

	// 解锁账号
	if reqBody.UnlockNow {
		if err := h.operationAccountService.UnlockAccount(c.Request.Context(), username); err != nil {
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

	dto := appAccount.ChangeUsernameDTO{
		OldUsername: oldUsername,
		NewUsername: strings.TrimSpace(reqBody.NewUsername),
	}

	if err := h.operationAccountService.ChangeUsername(c.Request.Context(), dto); err != nil {
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

	// 解析用户ID
	userID, err := parseUserID(reqBody.UserID)
	if err != nil {
		h.Error(c, err)
		return
	}

	ctx := c.Request.Context()

	// 检查是否已存在绑定
	appID := strings.TrimSpace(reqBody.AppID)
	openID := strings.TrimSpace(reqBody.OpenID)

	existingResult, err := h.wechatAccountService.GetByWeChatRef(ctx, openID, appID)
	created := false
	var accountID domain.AccountID

	if err == nil {
		// 已存在，检查是否属于同一用户
		if existingResult.Account.UserID != userID {
			h.ErrorWithCode(c, code.ErrInvalidArgument, "wechat binding already associated with another user")
			return
		}
		accountID = existingResult.Account.ID
	} else {
		// 不存在，创建新绑定
		if !perrors.IsCode(err, code.ErrDatabase) {
			h.Error(c, err)
			return
		}

		// 构建 DTO
		dto := appAccount.BindWeChatAccountDTO{
			AccountID:  accountID, // 需要先创建或查找账号
			ExternalID: openID,
			AppID:      appID,
			Nickname:   reqBody.Nickname,
			Avatar:     reqBody.Avatar,
		}

		metaBytes, err := reqBody.MetaJSON()
		if err != nil {
			h.Error(c, err)
			return
		}
		dto.Meta = metaBytes

		// TODO: 这里需要先确保账号存在，或者修改应用服务接口支持创建账号+绑定微信
		// 当前简化处理：直接绑定到已存在的账号
		if err := h.wechatAccountService.BindWeChatAccount(ctx, dto); err != nil {
			h.Error(c, err)
			return
		}
		created = true
	}

	// 更新微信资料（如果提供了）
	if reqBody.Nickname != nil || reqBody.Avatar != nil || reqBody.UnionID != nil {
		if reqBody.Nickname != nil || reqBody.Avatar != nil {
			metaBytes, _ := reqBody.MetaJSON()
			profileDTO := appAccount.UpdateWeChatProfileDTO{
				AccountID: accountID,
				Nickname:  reqBody.Nickname,
				Avatar:    reqBody.Avatar,
				Meta:      metaBytes,
			}
			if err := h.wechatAccountService.UpdateProfile(ctx, profileDTO); err != nil {
				h.Error(c, err)
				return
			}
		}

		if reqBody.UnionID != nil && strings.TrimSpace(*reqBody.UnionID) != "" {
			if err := h.wechatAccountService.SetUnionID(ctx, accountID, strings.TrimSpace(*reqBody.UnionID)); err != nil {
				h.Error(c, err)
				return
			}
		}
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, resp.NewBindResult(accountID, created))
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

	// 标准化可选字段
	nickname, _ := normalizeOptionalString(reqBody.Nickname)
	avatar, _ := normalizeOptionalString(reqBody.Avatar)

	metaBytes, err := reqBody.MetaJSON()
	if err != nil {
		h.Error(c, err)
		return
	}

	// 构建 DTO 并调用应用服务
	dto := appAccount.UpdateWeChatProfileDTO{
		AccountID: accountID,
		Nickname:  nickname,
		Avatar:    avatar,
		Meta:      metaBytes,
	}

	if err := h.wechatAccountService.UpdateProfile(c.Request.Context(), dto); err != nil {
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

	if err := h.wechatAccountService.SetUnionID(c.Request.Context(), accountID, strings.TrimSpace(reqBody.UnionID)); err != nil {
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

	result, err := h.accountService.GetAccountByID(c.Request.Context(), accountID)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, resp.NewAccount(result.Account))
}

// EnableAccount handles POST /v1/accounts/{accountId}:enable.
func (h *AccountHandler) EnableAccount(c *gin.Context) {
	accountID, err := parseAccountID(c.Param("accountId"))
	if err != nil {
		h.Error(c, err)
		return
	}

	if err := h.accountService.EnableAccount(c.Request.Context(), accountID); err != nil {
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

	if err := h.accountService.DisableAccount(c.Request.Context(), accountID); err != nil {
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

	accounts, err := h.accountService.ListAccountsByUserID(c.Request.Context(), userID)
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

	account, err := h.lookupService.FindByProvider(c.Request.Context(), provider, externalID, appID)
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

	result, err := h.operationAccountService.GetByUsername(c.Request.Context(), username)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, resp.NewOperationCredentialView(result.Account, result.OperationData))
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
