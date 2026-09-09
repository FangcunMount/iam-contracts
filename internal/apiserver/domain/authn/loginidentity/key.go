package loginidentity

import (
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// ProviderKey 用于解析登录身份的提供者键。
// Provider、Realm、Identifier 定位登录身份；GlobalIdentifier 提供可选的跨 Realm 关联。
// 例如：username:default:username、phone:global:+1234567890、wechat_minip:appid:openid:unionid、wecom:corp_id:userid。
type ProviderKey struct {
	// ---- 身份唯一定位 ----
	provider   Provider // 登录身份提供者
	realm      string   // Provider 内的身份命名空间
	identifier string   // Realm 内的登录标识

	// ---- 跨命名空间关联 ----
	globalIdentifier string // 可选的跨 Realm 标识
}

// newProviderKey 创建已经完成规范化和完整性校验的提供者键。
// 仅允许 Provider 专用构造器调用，避免调用方自由组合 Provider 与 Realm。
func newProviderKey(provider Provider, realm, identifier, globalIdentifier string) (ProviderKey, error) {
	realm = strings.TrimSpace(realm)
	identifier = strings.TrimSpace(identifier)
	globalIdentifier = strings.TrimSpace(globalIdentifier)
	if !provider.Validate() {
		return ProviderKey{}, perrors.WithCode(code.ErrInvalidArgument, "invalid login identity provider: %s", provider)
	}
	if realm == "" {
		return ProviderKey{}, perrors.WithCode(code.ErrInvalidArgument, "realm is required")
	}
	if identifier == "" {
		return ProviderKey{}, perrors.WithCode(code.ErrInvalidArgument, "identifier is required")
	}
	return ProviderKey{
		provider:         provider,
		realm:            realm,
		identifier:       identifier,
		globalIdentifier: globalIdentifier,
	}, nil
}

// NewUsernameProviderKey 创建默认命名空间的用户名登录身份键。
func NewUsernameProviderKey(username string) (ProviderKey, error) {
	return newProviderKey(ProviderUsername, RealmDefault, username, "")
}

// NewMockConsumerProviderKey 创建默认域模拟消费者登录身份键。
func NewMockConsumerProviderKey(username string) (ProviderKey, error) {
	return newProviderKey(ProviderUsername, RealmDefault, username, "")
}

// NewPhoneProviderKey 创建全局手机号登录身份键。
func NewPhoneProviderKey(phone meta.Phone) (ProviderKey, error) {
	return newProviderKey(ProviderPhone, RealmGlobal, phone.String(), "")
}

// NewWechatMinipProviderKey 创建微信小程序登录身份键。
func NewWechatMinipProviderKey(appID, openID, unionID string) (ProviderKey, error) {
	return newProviderKey(ProviderWechatMinip, appID, openID, unionID)
}

// NewWechatOpenProviderKey 创建微信开放平台登录身份键。
func NewWechatOpenProviderKey(appID, openID, unionID string) (ProviderKey, error) {
	return newProviderKey(ProviderWechatOpen, appID, openID, unionID)
}

// NewWecomProviderKey 创建企业微信登录身份键。
func NewWecomProviderKey(corpID, userIDInWecom string) (ProviderKey, error) {
	return newProviderKey(ProviderWecom, corpID, userIDInWecom, "")
}

// Provider 返回身份提供者。
func (k ProviderKey) Provider() Provider { return k.provider }

// Realm 返回 Provider 内的身份命名空间。
func (k ProviderKey) Realm() string { return k.realm }

// Identifier 返回 Realm 内的登录标识。
func (k ProviderKey) Identifier() string { return k.identifier }

// GlobalIdentifier 返回可选的跨 Realm 标识。
func (k ProviderKey) GlobalIdentifier() string { return k.globalIdentifier }

// IsValid 是否有效
func (k ProviderKey) IsValid() bool {
	return k.provider.Validate() &&
		strings.TrimSpace(k.realm) != "" &&
		strings.TrimSpace(k.identifier) != ""
}
