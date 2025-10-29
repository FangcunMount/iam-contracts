// Package handler 微信应用管理 REST API 处理器
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application/wechatapp"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/interface/restful/request"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/interface/restful/response"
)

// WechatAppHandler 微信应用管理 REST 处理器
//
// 依赖倒置原则：Handler 依赖应用服务接口，不依赖具体实现
type WechatAppHandler struct {
	*BaseHandler
	appService        wechatapp.WechatAppApplicationService
	credentialService wechatapp.WechatAppCredentialApplicationService
	tokenService      wechatapp.WechatAppTokenApplicationService
}

// NewWechatAppHandler 创建微信应用处理器
func NewWechatAppHandler(
	appService wechatapp.WechatAppApplicationService,
	credentialService wechatapp.WechatAppCredentialApplicationService,
	tokenService wechatapp.WechatAppTokenApplicationService,
) *WechatAppHandler {
	return &WechatAppHandler{
		BaseHandler:       NewBaseHandler(),
		appService:        appService,
		credentialService: credentialService,
		tokenService:      tokenService,
	}
}

// CreateWechatApp 创建微信应用
// @Summary 创建微信应用
// @Tags IDP-WechatApp
// @Accept json
// @Produce json
// @Param request body request.CreateWechatAppRequest true "创建微信应用请求"
// @Success 201 {object} response.WechatAppResponse "创建成功"
// @Failure 400 {object} response.ErrorResponse "请求参数错误"
// @Failure 500 {object} response.ErrorResponse "服务器内部错误"
// @Router /idp/wechat-apps [post]
func (h *WechatAppHandler) CreateWechatApp(c *gin.Context) {
	var req request.CreateWechatAppRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	// 转换为应用层 DTO
	dto := wechatapp.CreateWechatAppDTO{
		AppID:     req.AppID,
		Name:      req.Name,
		Type:      domain.AppType(req.Type),
		AppSecret: req.AppSecret,
	}

	// 调用应用服务
	result, err := h.appService.CreateApp(c.Request.Context(), dto)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 转换为 HTTP 响应
	resp := &response.WechatAppResponse{
		ID:     result.ID,
		AppID:  result.AppID,
		Name:   result.Name,
		Type:   string(result.Type),
		Status: string(result.Status),
	}

	h.Created(c, resp)
}

// GetWechatApp 查询微信应用
// @Summary 查询微信应用
// @Tags IDP-WechatApp
// @Accept json
// @Produce json
// @Param app_id path string true "微信应用 ID"
// @Success 200 {object} response.WechatAppResponse "查询成功"
// @Failure 404 {object} response.ErrorResponse "应用不存在"
// @Failure 500 {object} response.ErrorResponse "服务器内部错误"
// @Router /idp/wechat-apps/{app_id} [get]
func (h *WechatAppHandler) GetWechatApp(c *gin.Context) {
	var req request.GetWechatAppRequest
	if err := h.BindURI(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	// 调用应用服务
	result, err := h.appService.GetApp(c.Request.Context(), req.AppID)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 转换为 HTTP 响应
	resp := &response.WechatAppResponse{
		ID:     result.ID,
		AppID:  result.AppID,
		Name:   result.Name,
		Type:   string(result.Type),
		Status: string(result.Status),
	}

	h.Success(c, resp)
}

// RotateAuthSecret 轮换认证密钥
// @Summary 轮换认证密钥（AppSecret）
// @Tags IDP-WechatApp
// @Accept json
// @Produce json
// @Param request body request.RotateAuthSecretRequest true "轮换认证密钥请求"
// @Success 200 {object} response.RotateSecretResponse "轮换成功"
// @Failure 400 {object} response.ErrorResponse "请求参数错误"
// @Failure 404 {object} response.ErrorResponse "应用不存在"
// @Failure 500 {object} response.ErrorResponse "服务器内部错误"
// @Router /idp/wechat-apps/rotate-auth-secret [post]
func (h *WechatAppHandler) RotateAuthSecret(c *gin.Context) {
	var req request.RotateAuthSecretRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	// 调用应用服务
	err := h.credentialService.RotateAuthSecret(
		c.Request.Context(),
		req.AppID,
		req.NewSecret,
	)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 返回成功响应
	resp := &response.RotateSecretResponse{
		Success: true,
		Message: "Auth secret rotated successfully",
	}

	h.Success(c, resp)
}

// RotateMsgSecret 轮换消息密钥
// @Summary 轮换消息加解密密钥
// @Tags IDP-WechatApp
// @Accept json
// @Produce json
// @Param request body request.RotateMsgSecretRequest true "轮换消息密钥请求"
// @Success 200 {object} response.RotateSecretResponse "轮换成功"
// @Failure 400 {object} response.ErrorResponse "请求参数错误"
// @Failure 404 {object} response.ErrorResponse "应用不存在"
// @Failure 500 {object} response.ErrorResponse "服务器内部错误"
// @Router /idp/wechat-apps/rotate-msg-secret [post]
func (h *WechatAppHandler) RotateMsgSecret(c *gin.Context) {
	var req request.RotateMsgSecretRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	// 调用应用服务
	err := h.credentialService.RotateMsgSecret(
		c.Request.Context(),
		req.AppID,
		req.CallbackToken,
		req.EncodingAESKey,
	)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 返回成功响应
	resp := &response.RotateSecretResponse{
		Success: true,
		Message: "Message secret rotated successfully",
	}

	h.Success(c, resp)
}

// GetAccessToken 获取访问令牌
// @Summary 获取访问令牌（带缓存和自动刷新）
// @Tags IDP-WechatApp
// @Accept json
// @Produce json
// @Param app_id path string true "微信应用 ID"
// @Success 200 {object} response.AccessTokenResponse "获取成功"
// @Failure 404 {object} response.ErrorResponse "应用不存在"
// @Failure 500 {object} response.ErrorResponse "服务器内部错误"
// @Router /idp/wechat-apps/{app_id}/access-token [get]
func (h *WechatAppHandler) GetAccessToken(c *gin.Context) {
	var req request.GetAccessTokenRequest
	if err := h.BindURI(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	// 调用应用服务
	token, err := h.tokenService.GetAccessToken(c.Request.Context(), req.AppID)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 返回成功响应
	resp := &response.AccessTokenResponse{
		AccessToken: token,
		ExpiresIn:   7200, // 微信 access_token 默认 7200 秒
	}

	h.Success(c, resp)
}

// RefreshAccessToken 刷新访问令牌
// @Summary 强制刷新访问令牌
// @Tags IDP-WechatApp
// @Accept json
// @Produce json
// @Param request body request.RefreshAccessTokenRequest true "刷新访问令牌请求"
// @Success 200 {object} response.AccessTokenResponse "刷新成功"
// @Failure 400 {object} response.ErrorResponse "请求参数错误"
// @Failure 404 {object} response.ErrorResponse "应用不存在"
// @Failure 500 {object} response.ErrorResponse "服务器内部错误"
// @Router /idp/wechat-apps/refresh-access-token [post]
func (h *WechatAppHandler) RefreshAccessToken(c *gin.Context) {
	var req request.RefreshAccessTokenRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	// 调用应用服务
	token, err := h.tokenService.RefreshAccessToken(c.Request.Context(), req.AppID)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 返回成功响应
	resp := &response.AccessTokenResponse{
		AccessToken: token,
		ExpiresIn:   7200,
	}

	h.Success(c, resp)
}
