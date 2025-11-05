package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	appAccount "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/account"
	appRegister "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/register"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	req "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/interface/restful/request"
	resp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/interface/restful/response"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	_ "github.com/FangcunMount/iam-contracts/pkg/core"
)

// AccountHandler 账户管理 HTTP Handler
type AccountHandler struct {
	*BaseHandler
	accountService    appAccount.AccountApplicationService
	credentialService appAccount.CredentialApplicationService
	registerService   appRegister.RegisterApplicationService
}

// NewAccountHandler 创建账户处理器
func NewAccountHandler(
	accountService appAccount.AccountApplicationService,
	credentialService appAccount.CredentialApplicationService,
	registerService appRegister.RegisterApplicationService,
) *AccountHandler {
	return &AccountHandler{
		BaseHandler:       NewBaseHandler(),
		accountService:    accountService,
		credentialService: credentialService,
		registerService:   registerService,
	}
}

// GetAccountByID 根据账户ID获取账户信息
func (h *AccountHandler) GetAccountByID(c *gin.Context) {
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

	h.Success(c, toAccountResponse(result))
}

// RegisterWithWeChat 微信用户注册
func (h *AccountHandler) RegisterWithWeChat(c *gin.Context) {
	var reqBody req.RegisterWeChatAccountReq
	if err := h.BindJSON(c, &reqBody); err != nil {
		h.Error(c, err)
		return
	}

	if err := reqBody.Validate(); err != nil {
		h.Error(c, err)
		return
	}

	phone := meta.NewPhone(strings.TrimSpace(reqBody.Phone))
	var email meta.Email
	if strings.TrimSpace(reqBody.Email) != "" {
		email = meta.NewEmail(strings.TrimSpace(reqBody.Email))
	}

	appID := strings.TrimSpace(reqBody.AppID)
	openID := strings.TrimSpace(reqBody.OpenID)

	profile := make(map[string]string)
	if reqBody.Nickname != nil && *reqBody.Nickname != "" {
		profile["nickname"] = *reqBody.Nickname
	}
	if reqBody.Avatar != nil && *reqBody.Avatar != "" {
		profile["avatar"] = *reqBody.Avatar
	}

	metaMap, err := reqBody.MetaJSON()
	if err != nil {
		h.Error(c, err)
		return
	}

	registerReq := appRegister.RegisterRequest{
		Name:           strings.TrimSpace(reqBody.Name),
		Phone:          phone,
		Email:          email,
		CredentialType: appRegister.CredTypeWechat,
		WechatAppID:    &appID,
		WechatOpenID:   &openID,
		WechatUnionID:  reqBody.UnionID,
		Profile:        profile,
		Meta:           metaMap,
	}

	result, err := h.registerService.Register(c.Request.Context(), registerReq)
	if err != nil {
		h.Error(c, err)
		return
	}

	response := resp.RegisterResult{
		UserID:       result.UserID.String(),
		UserName:     result.UserName,
		Phone:        result.Phone.String(),
		Email:        result.Email.String(),
		AccountID:    result.AccountID.String(),
		AccountType:  string(result.AccountType),
		ExternalID:   string(result.ExternalID),
		CredentialID: result.CredentialID,
		IsNewUser:    result.IsNewUser,
		IsNewAccount: result.IsNewAccount,
	}

	h.Created(c, response)
}

// parseAccountID 解析账户ID
func parseAccountID(idStr string) (meta.ID, error) {
	idStr = strings.TrimSpace(idStr)
	if idStr == "" {
		return meta.ID{}, perrors.WithCode(code.ErrInvalidArgument, "accountId is required")
	}

	var id uint64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		return meta.ID{}, perrors.WithCode(code.ErrInvalidArgument, "invalid accountId format")
	}

	return meta.NewID(id), nil
}

// toAccountResponse 转换为账户响应（简化版）
func toAccountResponse(result *appAccount.AccountResult) resp.Account {
	appIDStr := string(result.AppID)
	return resp.Account{
		ID:         result.AccountID.String(),
		UserID:     result.UserID.String(),
		Provider:   string(result.Type),
		ExternalID: string(result.ExternalID),
		AppID:      &appIDStr,
		Status:     result.Status.String(),
	}
}

// UpdateProfile 更新账户资料
func (h *AccountHandler) UpdateProfile(c *gin.Context) {
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

	profile := make(map[string]string)
	if reqBody.Nickname != nil && *reqBody.Nickname != "" {
		profile["nickname"] = *reqBody.Nickname
	}
	if reqBody.Avatar != nil && *reqBody.Avatar != "" {
		profile["avatar"] = *reqBody.Avatar
	}

	// 将 Meta 转换为 map[string]string
	if reqBody.Meta != nil {
		for k, v := range reqBody.Meta {
			if str, ok := v.(string); ok {
				profile[k] = str
			}
		}
	}

	if err := h.accountService.UpdateProfile(c.Request.Context(), accountID, profile); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, resp.MessageResponse{Message: "Profile updated successfully"})
}

// SetUnionID 设置账户的 UnionID
func (h *AccountHandler) SetUnionID(c *gin.Context) {
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

	unionID := domain.UnionID(reqBody.UnionID)
	if err := h.accountService.SetUniqueID(c.Request.Context(), accountID, unionID); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, resp.MessageResponse{Message: "UnionID set successfully"})
}

// DisableAccount 禁用账户
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

	h.Success(c, resp.MessageResponse{Message: "Account disabled successfully"})
}

// EnableAccount 启用账户
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

	h.Success(c, resp.MessageResponse{Message: "Account enabled successfully"})
}

// GetCredentials 获取账户的所有凭证
func (h *AccountHandler) GetCredentials(c *gin.Context) {
	accountID, err := parseAccountID(c.Param("accountId"))
	if err != nil {
		h.Error(c, err)
		return
	}

	credentials, err := h.credentialService.GetCredentialsByAccountID(c.Request.Context(), accountID)
	if err != nil {
		h.Error(c, err)
		return
	}

	items := make([]resp.Credential, 0, len(credentials))
	for _, cred := range credentials {
		items = append(items, resp.Credential{
			ID:            cred.ID,
			AccountID:     cred.AccountID,
			Type:          string(cred.Type),
			IDP:           ptrToString(cred.IDP),
			IDPIdentifier: cred.IDPIdentifier,
			AppID:         ptrToString(cred.AppID),
			Status:        cred.Status.String(),
		})
	}

	h.Success(c, resp.CredentialList{
		Total: len(items),
		Items: items,
	})
}

// ptrToString 将字符串指针转为字符串，nil 返回空字符串
func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
