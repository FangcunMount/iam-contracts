package loginidentity

import (
	"time"

	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// LoginIdentity 绑定 IAM 用户到具体的登录标识
type LoginIdentity struct {
	// ---- 身份标识与归属 ----
	ID     meta.ID // 登录身份ID
	UserID meta.ID // 归属的 IAM 用户 ID

	// ---- 提供者身份定位 ----
	Provider         Provider // 登录身份提供者
	Realm            string   // Provider 内的身份命名空间，如 app_id、corp_id
	Identifier       string   // Realm 内的登录标识，与 Provider、Realm 共同组成唯一键
	GlobalIdentifier string   // 可选的跨 Realm 标识，如微信 unionid

	// ---- 状态与绑定记录 ----
	Status     Status     // 状态
	VerifiedAt *time.Time // 登录身份的验证时间，不是每次登录的认证时间
	LinkedAt   time.Time  // 绑定时间

	// ---- 附加资料 ----
	Profile map[string]string // 登录身份附带的资料
	Meta    map[string]string // 登录身份扩展元数据

	// ---- 记录时间 ----
	CreatedAt time.Time // 创建时间
	UpdatedAt time.Time // 更新时间
}

// UniqueKey 唯一键
func (i *LoginIdentity) UniqueKey() (Provider, string, string) {
	if i == nil {
		return "", "", ""
	}
	return i.Provider, i.Realm, i.Identifier
}

// IsActive 是否活动
func (i *LoginIdentity) IsActive() bool {
	return i != nil && i.Status == StatusActive
}

// VerifyAt 验证时间
func (i *LoginIdentity) VerifyAt(t time.Time) {
	i.VerifiedAt = &t
}

// Activate 激活
func (i *LoginIdentity) Activate() { i.Status = StatusActive }

// Disable 禁用
func (i *LoginIdentity) Disable() { i.Status = StatusDisabled }
