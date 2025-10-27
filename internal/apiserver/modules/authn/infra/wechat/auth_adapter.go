// Package wechat 微信认证适配器
package wechat

import (
	"context"
	"fmt"
	"net/http"

	wechatsdk "github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	miniconfig "github.com/silenceper/wechat/v2/miniprogram/config"
)

// AuthAdapter 微信认证适配器
//
// 实现 authentication.port.WeChatAuthPort 接口
// 调用微信 API 换取用户 openID
type AuthAdapter struct {
	httpClient *http.Client
	wechat     *wechatsdk.Wechat
	cache      cache.Cache
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
		if a.wechat != nil && client != nil {
			a.wechat.SetHTTPClient(client)
		}
	}
}

// WithCache 设置缓存实现
func WithCache(c cache.Cache) AuthAdapterOption {
	return func(a *AuthAdapter) {
		if c == nil {
			return
		}
		a.cache = c
		if a.wechat != nil {
			a.wechat.SetCache(c)
		}
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
		wechat:     wechatsdk.NewWechat(),
		cache:      cache.NewMemory(),
		appConfigs: make(map[string]string),
	}

	adapter.wechat.SetHTTPClient(adapter.httpClient)
	adapter.wechat.SetCache(adapter.cache)

	for _, opt := range opts {
		opt(adapter)
	}

	return adapter
}

// ExchangeOpenID 通过微信授权码换取 openID
func (a *AuthAdapter) ExchangeOpenID(ctx context.Context, code, appID string) (string, error) {
	// 获取 appSecret
	appSecret, ok := a.appConfigs[appID]
	if !ok {
		return "", fmt.Errorf("app config not found for appID: %s", appID)
	}

	cfg := &miniconfig.Config{
		AppID:     appID,
		AppSecret: appSecret,
		Cache:     a.cache,
	}

	mp := a.wechat.GetMiniProgram(cfg)
	auth := mp.GetAuth()
	session, err := auth.Code2SessionContext(ctx, code)
	if err != nil {
		return "", fmt.Errorf("failed to exchange openID from wechat: %w", err)
	}

	if session.OpenID == "" {
		return "", fmt.Errorf("openid is empty in wechat response")
	}

	return session.OpenID, nil
}
