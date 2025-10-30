package wechatapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	miniConfig "github.com/silenceper/wechat/v2/miniprogram/config"
)

// AuthProvider 微信认证服务（使用 silenceper SDK）
type AuthProvider struct {
	cache cache.Cache // SDK 使用的缓存
}

// NewAuthProvider 创建微信认证服务实例
func NewAuthProvider(cache cache.Cache) *AuthProvider {
	return &AuthProvider{
		cache: cache,
	}
}

// Code2SessionResult code2Session 返回结果
type Code2SessionResult struct {
	OpenID     string
	UnionID    string
	SessionKey string
}

// Code2Session 使用登录码进行登录
func (p *AuthProvider) Code2Session(ctx context.Context, appID, appSecret, jsCode string) (*Code2SessionResult, error) {
	if appID == "" || appSecret == "" {
		return nil, errors.New("appID and appSecret cannot be empty")
	}
	if jsCode == "" {
		return nil, errors.New("jsCode cannot be empty")
	}

	// 创建小程序实例
	wc := wechat.NewWechat()
	cfg := &miniConfig.Config{
		AppID:     appID,
		AppSecret: appSecret,
		Cache:     p.cache,
	}

	miniProgram := wc.GetMiniProgram(cfg)
	authAPI := miniProgram.GetAuth()

	// 调用 code2Session
	result, err := authAPI.Code2Session(jsCode)
	if err != nil {
		return nil, fmt.Errorf("failed to call code2session: %w", err)
	}

	// 检查返回值
	if result.OpenID == "" || result.SessionKey == "" {
		return nil, errors.New("empty openid or session_key returned")
	}

	return &Code2SessionResult{
		OpenID:     result.OpenID,
		UnionID:    result.UnionID,
		SessionKey: result.SessionKey,
	}, nil
}

// DecryptPhoneResult 解密手机号返回结果
type DecryptPhoneResult struct {
	PhoneNumber     string
	PurePhoneNumber string
	CountryCode     string
}

// DecryptPhone 解密用户手机号
func (p *AuthProvider) DecryptPhone(ctx context.Context, appID, appSecret, sessionKey, encryptedData, iv string) (*DecryptPhoneResult, error) {
	if appID == "" || appSecret == "" {
		return nil, errors.New("appID and appSecret cannot be empty")
	}
	if sessionKey == "" || encryptedData == "" || iv == "" {
		return nil, errors.New("sessionKey, encryptedData and iv cannot be empty")
	}

	// 创建小程序实例
	wc := wechat.NewWechat()
	cfg := &miniConfig.Config{
		AppID:     appID,
		AppSecret: appSecret,
		Cache:     p.cache,
	}

	miniProgram := wc.GetMiniProgram(cfg)
	encryptor := miniProgram.GetEncryptor()

	// 解密数据
	plainData, err := encryptor.Decrypt(sessionKey, encryptedData, iv)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	// 解析手机号
	type PhoneInfo struct {
		PhoneNumber     string `json:"phoneNumber"`
		PurePhoneNumber string `json:"purePhoneNumber"`
		CountryCode     string `json:"countryCode"`
	}

	var phoneInfo PhoneInfo
	if err := json.Unmarshal([]byte(plainData.PhoneNumber), &phoneInfo); err != nil {
		// 如果解析失败，尝试直接使用 PhoneNumber 字段
		return &DecryptPhoneResult{
			PhoneNumber: plainData.PhoneNumber,
		}, nil
	}

	return &DecryptPhoneResult{
		PhoneNumber:     phoneInfo.PhoneNumber,
		PurePhoneNumber: phoneInfo.PurePhoneNumber,
		CountryCode:     phoneInfo.CountryCode,
	}, nil
}
