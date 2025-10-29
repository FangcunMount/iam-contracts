package handler

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/login"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/application/token"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/interface/restful/request"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/interface/restful/response"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	_ "github.com/FangcunMount/iam-contracts/pkg/core" // imported for swagger
)

// AuthHandler 认证 HTTP 处理器
type AuthHandler struct {
	*BaseHandler
	loginService *login.LoginService
	tokenService *token.TokenService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(loginService *login.LoginService, tokenService *token.TokenService) *AuthHandler {
	return &AuthHandler{
		BaseHandler:  NewBaseHandler(),
		loginService: loginService,
		tokenService: tokenService,
	}
}

// Login 统一登录端点（符合 API 文档）
// @Summary 登录
// @Description 使用不同的认证方式进行登录（basic: 用户名密码, wx:minip: 微信小程序）
// @Tags Authentication-Auth
// @Accept json
// @Produce json
// @Param request body request.LoginRequest true "登录请求"
// @Success 200 {object} response.TokenPair "登录成功"
// @Failure 400 {object} core.ErrResponse "参数错误"
// @Failure 401 {object} core.ErrResponse "认证失败"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req request.LoginRequest

	// 绑定请求参数
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	// 获取客户端 IP
	clientIP := c.ClientIP()

	// 根据认证方式分发
	var result *login.LoginWithPasswordResponse
	var err error

	switch req.Method {
	case "basic":
		result, err = h.handleBasicLogin(c, req.Credentials, clientIP, req.DeviceID)
	case "wx:minip":
		result, err = h.handleWeChatMiniLogin(c, req.Credentials, clientIP, req.DeviceID)
	default:
		h.Error(c, perrors.WithCode(code.ErrInvalidArgument, "unsupported authentication method: %s", req.Method))
		return
	}

	if err != nil {
		h.Error(c, err)
		return
	}

	// 构造符合 API 文档的响应
	tokenPair := &response.TokenPair{
		AccessToken:  result.AccessToken,
		TokenType:    result.TokenType,
		ExpiresIn:    result.ExpiresIn,
		RefreshToken: result.RefreshToken,
	}

	h.Success(c, tokenPair)
}

// handleBasicLogin 处理基本认证（用户名密码）
func (h *AuthHandler) handleBasicLogin(c *gin.Context, credentials json.RawMessage, ip, deviceID string) (*login.LoginWithPasswordResponse, error) {
	var creds request.BasicCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, perrors.WithCode(code.ErrBind, "invalid basic credentials: %v", err)
	}

	// 调用应用服务
	return h.loginService.LoginWithPassword(c.Request.Context(), &login.LoginWithPasswordRequest{
		Username: creds.Username,
		Password: creds.Password,
		IP:       ip,
		Device:   deviceID,
	})
}

// handleWeChatMiniLogin 处理微信小程序登录
func (h *AuthHandler) handleWeChatMiniLogin(c *gin.Context, credentials json.RawMessage, ip, deviceID string) (*login.LoginWithPasswordResponse, error) {
	var creds request.WeChatMiniCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, perrors.WithCode(code.ErrBind, "invalid wechat credentials: %v", err)
	}

	// 调用应用服务（返回值类型暂时复用，后续可以统一）
	result, err := h.loginService.LoginWithWeChat(c.Request.Context(), &login.LoginWithWeChatRequest{
		Code:   creds.JSCode,
		AppID:  creds.AppID,
		IP:     ip,
		Device: deviceID,
	})
	if err != nil {
		return nil, err
	}

	// 转换为统一的响应类型
	return &login.LoginWithPasswordResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    result.TokenType,
		ExpiresIn:    result.ExpiresIn,
	}, nil
}

// RefreshToken 刷新令牌（符合 API 文档：POST /v1/auth/token）
// @Summary 刷新令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags Authentication-Tokens
// @Accept json
// @Produce json
// @Param request body request.RefreshTokenRequest true "刷新令牌请求"
// @Success 200 {object} response.TokenPair "刷新成功"
// @Failure 400 {object} core.ErrResponse "参数错误"
// @Failure 401 {object} core.ErrResponse "令牌无效"
// @Router /auth/token [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req request.RefreshTokenRequest

	// 绑定请求参数
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	// 调用应用服务
	result, err := h.tokenService.RefreshToken(c.Request.Context(), &token.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		h.Error(c, err)
		return
	}

	// 构造符合 API 文档的响应
	tokenPair := &response.TokenPair{
		AccessToken:  result.AccessToken,
		TokenType:    result.TokenType,
		ExpiresIn:    result.ExpiresIn,
		RefreshToken: result.RefreshToken,
	}

	h.Success(c, tokenPair)
}

// Logout 登出（符合 API 文档）
// @Summary 登出
// @Description 撤销访问令牌和刷新令牌
// @Tags Authentication-Tokens
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer {access_token}"
// @Param request body request.LogoutRequest false "登出请求"
// @Success 200 {object} map[string]interface{} "登出成功"
// @Failure 401 {object} core.ErrResponse "未授权"
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// 从 Header 中提取访问令牌
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		h.Error(c, perrors.WithCode(code.ErrUnauthenticated, "missing authorization header"))
		return
	}

	// 提取 Bearer token
	accessToken := strings.TrimPrefix(authHeader, "Bearer ")
	if accessToken == authHeader {
		h.Error(c, perrors.WithCode(code.ErrUnauthenticated, "invalid authorization header format"))
		return
	}

	// 绑定请求参数
	var req request.LogoutRequest
	_ = h.BindJSON(c, &req) // 忽略错误，因为参数是可选的

	// 处理 all 参数（撤销当前用户所有令牌）
	if req.All {
		// TODO: 实现撤销当前用户所有令牌的逻辑
		h.Error(c, perrors.WithCode(code.ErrInvalidArgument, "logout all tokens not implemented yet"))
		return
	}

	// 调用应用服务
	if err := h.tokenService.Logout(c.Request.Context(), &token.LogoutRequest{
		AccessToken:  accessToken,
		RefreshToken: req.RefreshToken,
	}); err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, gin.H{"message": "logout successful"})
}

// VerifyToken 验证令牌（符合 API 文档：POST /v1/auth/verify）
// @Summary 验证令牌
// @Description 验证访问令牌的有效性（验签 + 载荷校验 + 黑名单检查）
// @Tags Authentication-Tokens
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.VerifyTokenRequest false "验证令牌请求（可选）"
// @Success 200 {object} response.VerifyResponse "验证结果"
// @Failure 401 {object} core.ErrResponse "未授权"
// @Router /auth/verify [post]
func (h *AuthHandler) VerifyToken(c *gin.Context) {
	var req request.VerifyTokenRequest

	// 绑定请求参数（可选）
	_ = h.BindJSON(c, &req)

	// 如果请求体中没有提供 token，从 Header 获取
	accessToken := req.Token
	if accessToken == "" {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			accessToken = strings.TrimPrefix(authHeader, "Bearer ")
			if accessToken == authHeader {
				h.Error(c, perrors.WithCode(code.ErrUnauthenticated, "invalid authorization header format"))
				return
			}
		}
	}

	if accessToken == "" {
		h.Error(c, perrors.WithCode(code.ErrUnauthenticated, "missing access token"))
		return
	}

	// 调用应用服务验证令牌
	result, err := h.tokenService.VerifyToken(c.Request.Context(), &token.VerifyTokenRequest{
		AccessToken: accessToken,
	})
	if err != nil {
		h.Error(c, err)
		return
	}

	// 构造符合 API 文档的响应
	verifyResp := &response.VerifyResponse{
		Claims: response.TokenClaims{
			Sub: fmt.Sprintf("%d", result.UserID),
			AID: fmt.Sprintf("%d", result.AccountID),
			JTI: result.TokenID,
			// TODO: 补充其他字段（需要从令牌中解析）
		},
		Header:  map[string]interface{}{}, // TODO: 填充 JWT Header
		Blocked: !result.Valid,            // 如果无效则视为被拉黑
	}

	h.Success(c, verifyResp)
}

// GetJWKS 获取 JWKS 公钥集（符合 API 文档：GET /.well-known/jwks.json）
// @Summary 获取 JWKS
// @Description 获取用于验证 JWT 签名的公钥集
// @Tags Authentication-JWKS
// @Produce json
// @Success 200 {object} response.JWKSet "公钥集"
// @Router /.well-known/jwks.json [get]
func (h *AuthHandler) GetJWKS(c *gin.Context) {
	// TODO: 实现从配置或密钥管理服务获取公钥
	// 目前返回空集合
	jwks := &response.JWKSet{
		Keys: []response.JWK{},
	}

	h.Success(c, jwks)
}
