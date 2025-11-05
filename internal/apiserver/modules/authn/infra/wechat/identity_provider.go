package wechat

import (
	"context"
	"fmt"

	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	miniConfig "github.com/silenceper/wechat/v2/miniprogram/config"
	workConfig "github.com/silenceper/wechat/v2/work/config"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port"
)

// IdentityProviderImpl 微信身份提供商的实现（使用 silenceper SDK）
type IdentityProviderImpl struct {
	cache cache.Cache // SDK 使用的缓存
	// 微信小程序配置：appID -> appSecret
	minipSecrets map[string]string
	// 企业微信配置：corpID -> WecomConfig
	wecomSecrets map[string]WecomConfig
}

// WecomConfig 企业微信配置
type WecomConfig struct {
	AgentID string
	Secret  string
}

// 确保实现了接口
var _ port.IdentityProvider = (*IdentityProviderImpl)(nil)

// NewIdentityProvider 创建微信身份提供商
func NewIdentityProvider(
	cache cache.Cache,
	minipSecrets map[string]string,
	wecomSecrets map[string]WecomConfig,
) port.IdentityProvider {
	return &IdentityProviderImpl{
		cache:        cache,
		minipSecrets: minipSecrets,
		wecomSecrets: wecomSecrets,
	}
}

// ExchangeWxMinipCode 微信小程序 jsCode 换取 session
// 文档: https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-login/code2Session.html
func (p *IdentityProviderImpl) ExchangeWxMinipCode(ctx context.Context, appID, jsCode string) (openID, unionID string, err error) {
	// 获取 appSecret
	appSecret, ok := p.minipSecrets[appID]
	if !ok {
		return "", "", fmt.Errorf("wx minip app secret not found for appID: %s", appID)
	}

	// 创建小程序实例
	cfg := &miniConfig.Config{
		AppID:     appID,
		AppSecret: appSecret,
		Cache:     p.cache,
	}
	miniProgram := wechat.NewWechat().GetMiniProgram(cfg)

	// 调用 code2Session
	result, err := miniProgram.GetAuth().Code2Session(jsCode)
	if err != nil {
		return "", "", fmt.Errorf("failed to call code2session: %w", err)
	}

	// 检查返回值
	if result.OpenID == "" {
		return "", "", fmt.Errorf("openid is empty in code2session result")
	}

	return result.OpenID, result.UnionID, nil
}

// ExchangeWecomCode 企业微信 code 换取用户信息
// 文档: https://developer.work.weixin.qq.com/document/path/91023
func (p *IdentityProviderImpl) ExchangeWecomCode(ctx context.Context, corpID, code string) (openUserID, userID string, err error) {
	// 获取企业微信配置
	config, ok := p.wecomSecrets[corpID]
	if !ok {
		return "", "", fmt.Errorf("wecom config not found for corpID: %s", corpID)
	}

	// 创建企业微信实例
	cfg := &workConfig.Config{
		CorpID:     corpID,
		CorpSecret: config.Secret,
		AgentID:    config.AgentID,
		Cache:      p.cache,
	}
	workApp := wechat.NewWechat().GetWork(cfg)

	// 获取用户信息
	userInfo, err := workApp.GetOauth().GetUserInfo(code)
	if err != nil {
		return "", "", fmt.Errorf("failed to get wecom user info: %w", err)
	}

	// 企业成员返回 UserID，非企业成员返回 OpenID
	// UserID: 成员UserID（当是企业成员时返回）
	// OpenID: 非企业成员的标识（当是非企业成员时返回）
	// DeviceID: 手机设备号
	// ExternalUserID: 外部联系人ID

	return userInfo.OpenID, userInfo.UserID, nil
}
