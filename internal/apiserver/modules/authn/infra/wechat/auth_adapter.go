// Package wechat 微信认证适配器
package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// AuthAdapter 微信认证适配器
//
// 实现 authentication.port.WeChatAuthPort 接口
// 调用微信 API 换取用户 openID
type AuthAdapter struct {
	httpClient *http.Client
	// 微信小程序配置可以从配置文件或数据库读取
	// 这里简化处理，实际应该根据 appID 查询对应的 appSecret
	appConfigs map[string]string // appID -> appSecret
}

// AuthAdapterOption 微信认证适配器选项
type AuthAdapterOption func(*AuthAdapter)

// WithHTTPClient 设置 HTTP 客户端
func WithHTTPClient(client *http.Client) AuthAdapterOption {
	return func(a *AuthAdapter) {
		a.httpClient = client
	}
}

// WithAppConfig 添加应用配置
func WithAppConfig(appID, appSecret string) AuthAdapterOption {
	return func(a *AuthAdapter) {
		if a.appConfigs == nil {
			a.appConfigs = make(map[string]string)
		}
		a.appConfigs[appID] = appSecret
	}
}

// NewAuthAdapter 创建微信认证适配器
func NewAuthAdapter(opts ...AuthAdapterOption) *AuthAdapter {
	adapter := &AuthAdapter{
		httpClient: http.DefaultClient,
		appConfigs: make(map[string]string),
	}

	for _, opt := range opts {
		opt(adapter)
	}

	return adapter
}

// WeChatAuthResponse 微信认证响应
type WeChatAuthResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// ExchangeOpenID 通过微信授权码换取 openID
func (a *AuthAdapter) ExchangeOpenID(ctx context.Context, code, appID string) (string, error) {
	// 获取 appSecret
	appSecret, ok := a.appConfigs[appID]
	if !ok {
		return "", fmt.Errorf("app config not found for appID: %s", appID)
	}

	// 构建请求 URL
	// 文档: https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/login/auth.code2Session.html
	apiURL := "https://api.weixin.qq.com/sns/jscode2session"
	params := url.Values{}
	params.Set("appid", appID)
	params.Set("secret", appSecret)
	params.Set("js_code", code)
	params.Set("grant_type", "authorization_code")

	requestURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to wechat: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var wxResp WeChatAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&wxResp); err != nil {
		return "", fmt.Errorf("failed to decode wechat response: %w", err)
	}

	// 检查错误码
	if wxResp.ErrCode != 0 {
		return "", fmt.Errorf("wechat api error: code=%d, msg=%s", wxResp.ErrCode, wxResp.ErrMsg)
	}

	if wxResp.OpenID == "" {
		return "", fmt.Errorf("openid is empty in wechat response")
	}

	return wxResp.OpenID, nil
}
