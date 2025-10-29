// Package handler 微信认证 REST API 处理器
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/application/wechatsession"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/interface/restful/request"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/interface/restful/response"
)

// WechatAuthHandler 微信认证 REST 处理器
//
// 依赖倒置原则：Handler 依赖应用服务接口，不依赖具体实现
type WechatAuthHandler struct {
	*BaseHandler
	authService wechatsession.WechatAuthApplicationService
}

// NewWechatAuthHandler 创建微信认证处理器
func NewWechatAuthHandler(
	authService wechatsession.WechatAuthApplicationService,
) *WechatAuthHandler {
	return &WechatAuthHandler{
		BaseHandler: NewBaseHandler(),
		authService: authService,
	}
}

// LoginWithCode 使用微信登录码进行登录
// @Summary 微信小程序登录
// @Description 使用微信小程序登录码（wx.login 获取）进行用户认证，返回用户身份信息和会话密钥
// @Tags IDP-WechatAuth
// @Accept json
// @Produce json
// @Param request body request.LoginWithCodeRequest true "微信登录请求"
// @Success 200 {object} response.LoginResponse "登录成功"
// @Failure 400 {object} response.ErrorResponse "请求参数错误"
// @Failure 401 {object} response.ErrorResponse "登录失败（code 无效或已过期）"
// @Failure 500 {object} response.ErrorResponse "服务器内部错误"
// @Router /api/v1/idp/wechat/login [post]
func (h *WechatAuthHandler) LoginWithCode(c *gin.Context) {
	var req request.LoginWithCodeRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	// 转换为应用层 DTO
	dto := wechatsession.LoginWithCodeDTO{
		AppID:  req.AppID,
		JSCode: req.JSCode,
	}

	// 调用应用服务
	result, err := h.authService.LoginWithCode(c.Request.Context(), dto)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 转换为 HTTP 响应
	resp := &response.LoginResponse{
		Provider:    result.Provider,
		AppID:       result.AppID,
		OpenID:      result.OpenID,
		UnionID:     result.UnionID,
		DisplayName: result.DisplayName,
		AvatarURL:   result.AvatarURL,
		Phone:       result.Phone,
		Email:       result.Email,
		ExpiresIn:   result.ExpiresInSec,
		SessionKey:  result.SessionKey,
		Version:     result.Version,
	}

	h.Success(c, resp)
}

// DecryptUserPhone 解密用户手机号
// @Summary 解密微信用户手机号
// @Description 使用 session_key 解密微信小程序获取的加密手机号信息
// @Tags IDP-WechatAuth
// @Accept json
// @Produce json
// @Param request body request.DecryptPhoneRequest true "解密手机号请求"
// @Success 200 {object} response.DecryptPhoneResponse "解密成功"
// @Failure 400 {object} response.ErrorResponse "请求参数错误"
// @Failure 401 {object} response.ErrorResponse "解密失败（session_key 无效）"
// @Failure 500 {object} response.ErrorResponse "服务器内部错误"
// @Router /api/v1/idp/wechat/decrypt-phone [post]
func (h *WechatAuthHandler) DecryptUserPhone(c *gin.Context) {
	var req request.DecryptPhoneRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	// 转换为应用层 DTO
	dto := wechatsession.DecryptPhoneDTO{
		AppID:         req.AppID,
		OpenID:        req.OpenID,
		EncryptedData: req.EncryptedData,
		IV:            req.IV,
	}

	// 调用应用服务
	phone, err := h.authService.DecryptUserPhone(c.Request.Context(), dto)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 返回成功响应
	resp := &response.DecryptPhoneResponse{
		Phone: phone,
	}

	h.Success(c, resp)
}
