package handler

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/login"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/token"
	domainToken "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/token"
	req "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/interface/restful/request"
	resp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/interface/restful/response"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// AuthHandler 认证 HTTP 处理器
type AuthHandler struct {
	*BaseHandler
	loginService login.LoginApplicationService
	tokenService token.TokenApplicationService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(
	loginService login.LoginApplicationService,
	tokenService token.TokenApplicationService,
) *AuthHandler {
	return &AuthHandler{
		BaseHandler:  NewBaseHandler(),
		loginService: loginService,
		tokenService: tokenService,
	}
}

// Login 统一登录端点
func (h *AuthHandler) Login(c *gin.Context) {
	var reqBody req.LoginRequest
	if err := h.BindJSON(c, &reqBody); err != nil {
		h.Error(c, err)
		return
	}

	if err := reqBody.Validate(); err != nil {
		h.Error(c, err)
		return
	}

	// 根据认证方法路由到对应的处理函数
	switch reqBody.Method {
	case "password":
		h.handlePasswordLogin(c, reqBody)
	case "phone_otp":
		h.handlePhoneOTPLogin(c, reqBody)
	case "wechat":
		h.handleWeChatLogin(c, reqBody)
	case "wecom":
		h.handleWeComLogin(c, reqBody)
	default:
		h.Error(c, perrors.WithCode(code.ErrInvalidArgument, "unsupported authentication method: %s", reqBody.Method))
	}
}

// handlePasswordLogin 处理密码登录
func (h *AuthHandler) handlePasswordLogin(c *gin.Context, reqBody req.LoginRequest) {
	var creds req.PasswordCredentials
	if err := json.Unmarshal(reqBody.Credentials, &creds); err != nil {
		h.Error(c, perrors.WithCode(code.ErrBind, "invalid password credentials: %v", err))
		return
	}

	loginReq := login.LoginRequest{
		AuthType: login.AuthTypePassword,
		Username: &creds.Username,
		Password: &creds.Password,
	}
	if creds.TenantID != nil {
		tenantID := int64(*creds.TenantID)
		loginReq.TenantID = &tenantID
	}

	h.executeLogin(c, loginReq)
}

// handlePhoneOTPLogin 处理手机验证码登录
func (h *AuthHandler) handlePhoneOTPLogin(c *gin.Context, reqBody req.LoginRequest) {
	var creds req.PhoneOTPCredentials
	if err := json.Unmarshal(reqBody.Credentials, &creds); err != nil {
		h.Error(c, perrors.WithCode(code.ErrBind, "invalid phone OTP credentials: %v", err))
		return
	}

	loginReq := login.LoginRequest{
		AuthType:  login.AuthTypePhoneOTP,
		PhoneE164: &creds.Phone,
		OTPCode:   &creds.OTPCode,
	}

	h.executeLogin(c, loginReq)
}

// handleWeChatLogin 处理微信小程序登录
func (h *AuthHandler) handleWeChatLogin(c *gin.Context, reqBody req.LoginRequest) {
	var creds req.WeChatCredentials
	if err := json.Unmarshal(reqBody.Credentials, &creds); err != nil {
		h.Error(c, perrors.WithCode(code.ErrBind, "invalid wechat credentials: %v", err))
		return
	}

	loginReq := login.LoginRequest{
		AuthType:     login.AuthTypeWechat,
		WechatAppID:  &creds.AppID,
		WechatJSCode: &creds.Code,
	}

	h.executeLogin(c, loginReq)
}

// handleWeComLogin 处理企业微信登录
func (h *AuthHandler) handleWeComLogin(c *gin.Context, reqBody req.LoginRequest) {
	var creds req.WeComCredentials
	if err := json.Unmarshal(reqBody.Credentials, &creds); err != nil {
		h.Error(c, perrors.WithCode(code.ErrBind, "invalid wecom credentials: %v", err))
		return
	}

	loginReq := login.LoginRequest{
		AuthType:    login.AuthTypeWecom,
		WecomCorpID: &creds.CorpID,
		WecomCode:   &creds.AuthCode,
	}

	h.executeLogin(c, loginReq)
}

// executeLogin 执行登录并返回令牌
func (h *AuthHandler) executeLogin(c *gin.Context, loginReq login.LoginRequest) {
	result, err := h.loginService.Login(c.Request.Context(), loginReq)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 转换为 HTTP 响应格式
	tokenPair := h.convertTokenPair(result.TokenPair)
	h.Success(c, tokenPair)
}

// Logout 登出
func (h *AuthHandler) Logout(c *gin.Context) {
	var reqBody req.LogoutRequest
	if err := h.BindJSON(c, &reqBody); err != nil {
		h.Error(c, err)
		return
	}

	if err := reqBody.Validate(); err != nil {
		h.Error(c, err)
		return
	}

	logoutReq := login.LogoutRequest{
		AccessToken:  reqBody.AccessToken,
		RefreshToken: &reqBody.RefreshToken,
	}

	if err := h.loginService.Logout(c.Request.Context(), logoutReq); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, resp.MessageResponse{Message: "Logout successful"})
}

// RefreshToken 刷新访问令牌
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var reqBody req.RefreshTokenRequest
	if err := h.BindJSON(c, &reqBody); err != nil {
		h.Error(c, err)
		return
	}

	if err := reqBody.Validate(); err != nil {
		h.Error(c, err)
		return
	}

	result, err := h.tokenService.RefreshToken(c.Request.Context(), reqBody.RefreshToken)
	if err != nil {
		h.Error(c, err)
		return
	}

	tokenPair := h.convertTokenPair(result.TokenPair)
	h.Success(c, tokenPair)
}

// VerifyToken 验证访问令牌
func (h *AuthHandler) VerifyToken(c *gin.Context) {
	var reqBody req.VerifyTokenRequest
	if err := h.BindJSON(c, &reqBody); err != nil {
		h.Error(c, err)
		return
	}

	if err := reqBody.Validate(); err != nil {
		h.Error(c, err)
		return
	}

	result, err := h.tokenService.VerifyToken(c.Request.Context(), reqBody.AccessToken)
	if err != nil {
		h.Error(c, err)
		return
	}

	response := resp.TokenVerifyResponse{
		Valid:  result.Valid,
		Claims: nil,
	}

	if result.Valid && result.Claims != nil {
		response.Claims = &resp.TokenClaims{
			UserID:    result.Claims.UserID.String(),
			AccountID: result.Claims.AccountID.String(),
			TenantID:  nil, // Domain TokenClaims 没有 TenantID 字段
			Issuer:    "",  // Domain TokenClaims 没有 Issuer 字段
			IssuedAt:  result.Claims.IssuedAt,
			ExpiresAt: result.Claims.ExpiresAt,
			JTI:       result.Claims.TokenID,
		}
	}

	h.Success(c, response)
}

// RevokeToken 撤销访问令牌
func (h *AuthHandler) RevokeToken(c *gin.Context) {
	var reqBody req.RevokeTokenRequest
	if err := h.BindJSON(c, &reqBody); err != nil {
		h.Error(c, err)
		return
	}

	if err := reqBody.Validate(); err != nil {
		h.Error(c, err)
		return
	}

	if err := h.tokenService.RevokeToken(c.Request.Context(), reqBody.AccessToken); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, resp.MessageResponse{Message: "Token revoked successfully"})
}

// RevokeRefreshToken 撤销刷新令牌
func (h *AuthHandler) RevokeRefreshToken(c *gin.Context) {
	var reqBody req.RevokeRefreshTokenRequest
	if err := h.BindJSON(c, &reqBody); err != nil {
		h.Error(c, err)
		return
	}

	if err := reqBody.Validate(); err != nil {
		h.Error(c, err)
		return
	}

	if err := h.tokenService.RevokeRefreshToken(c.Request.Context(), reqBody.RefreshToken); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, resp.MessageResponse{Message: "Refresh token revoked successfully"})
}

// convertTokenPair 转换令牌对为 HTTP 响应格式
func (h *AuthHandler) convertTokenPair(tokenPair *domainToken.TokenPair) *resp.TokenPair {
	response := &resp.TokenPair{
		TokenType: "Bearer",
	}

	if tokenPair == nil {
		return response
	}

	if tokenPair.AccessToken != nil {
		response.AccessToken = tokenPair.AccessToken.Value
		response.ExpiresIn = int64(time.Until(tokenPair.AccessToken.ExpiresAt).Seconds())
	}

	if tokenPair.RefreshToken != nil {
		response.RefreshToken = tokenPair.RefreshToken.Value
	}

	return response
}
