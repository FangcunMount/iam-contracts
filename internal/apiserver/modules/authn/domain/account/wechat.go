package account

// WeChatAccount 微信账号（小程序/公众号）
type WeChatAccount struct {
	AccountID AccountID
	AppID     string  // 微信公众号/小程序 appid
	OpenID    string  // 普通用户的标识，对当前公众号/小程序唯一
	UnionID   *string // 用户在开放平台的唯一标识符
	Nickname  *string // 用户昵称
	AvatarURL *string // 用户头像，最后一个数值代表正方形头像大小（有0、46、64、96、132数值可选，0代表640*640正方形头像），用户没有头像时该项为空
	Meta      []byte  // 其他原始数据
}

func NewWeChatAccount(id AccountID, appid, openid string, opts ...WeChatAccountOption) WeChatAccount {
	account := WeChatAccount{
		AccountID: id,
		AppID:     appid,
		OpenID:    openid,
	}
	for _, opt := range opts {
		opt(&account)
	}
	return account
}

// WeChatAccountOption 微信账号选项
type WeChatAccountOption func(*WeChatAccount)

func WithWeChatAccountID(id AccountID) WeChatAccountOption {
	return func(a *WeChatAccount) { a.AccountID = id }
}
func WithWeChatUnionID(unionid string) WeChatAccountOption {
	return func(a *WeChatAccount) { a.UnionID = &unionid }
}
func WithWeChatNickname(nickname string) WeChatAccountOption {
	return func(a *WeChatAccount) { a.Nickname = &nickname }
}
func WithWeChatAvatarURL(avatar string) WeChatAccountOption {
	return func(a *WeChatAccount) { a.AvatarURL = &avatar }
}
func WithWeChatMeta(meta []byte) WeChatAccountOption {
	return func(a *WeChatAccount) { a.Meta = meta }
}

// UpdateMeta 更新 Meta 信息
func (a *WeChatAccount) UpdateMeta(meta []byte) {
	a.Meta = meta
}

// UpdateNickname 更新昵称
func (a *WeChatAccount) UpdateNickname(nickname string) {
	a.Nickname = &nickname
}

// UpdateAvatarURL 更新头像 URL
func (a *WeChatAccount) UpdateAvatarURL(avatar string) {
	a.AvatarURL = &avatar
}
