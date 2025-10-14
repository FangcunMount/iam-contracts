package account

// Account 代表第三方登录账号
type Account struct {
	ID         AccountID
	UserID     UserID   // 用户标识(来自用户中心)
	Provider   Provider // op:password | wx:minip | wecom:qr
	ExternalID string   // username | openid | open_userid(userid)
	AppID      *string  // 微信小程序 appid | 企业微信 corpid
	Status     AccountStatus
}

// NewAccount 创建第三方登录账号
func NewAccount(userID UserID, provider Provider, opts ...AccountOption) Account {
	account := Account{
		UserID:   userID,
		Provider: provider,
	}
	for _, opt := range opts {
		opt(&account)
	}
	return account
}

// AccountOption 第三方登录账号选项
type AccountOption func(*Account)

func WithID(id AccountID) AccountOption             { return func(a *Account) { a.ID = id } }
func WithExternalID(eid string) AccountOption       { return func(a *Account) { a.ExternalID = eid } }
func WithAppID(appid string) AccountOption          { return func(a *Account) { a.AppID = &appid } }
func WithStatus(status AccountStatus) AccountOption { return func(a *Account) { a.Status = status } }

// 状态变更方法
func (a *Account) Activate() { a.Status = StatusActive }
func (a *Account) Disable()  { a.Status = StatusDisabled }
func (a *Account) Archive()  { a.Status = StatusArchived }
func (a *Account) Delete()   { a.Status = StatusDeleted }

// 状态检查方法
func (a *Account) IsActive() bool   { return a.Status == StatusActive }
func (a *Account) IsDisabled() bool { return a.Status == StatusDisabled }
func (a *Account) IsArchived() bool { return a.Status == StatusArchived }
func (a *Account) IsDeleted() bool  { return a.Status == StatusDeleted }
