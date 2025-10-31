package account

import (
	"time"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	base "github.com/FangcunMount/iam-contracts/internal/pkg/database/mysql"
	"gorm.io/gorm"
)

// AccountPO 持久化对象，对应认证账号表。
type AccountPO struct {
	base.AuditFields
	UserID     idutil.ID `gorm:"column:user_id;type:bigint unsigned;not null;index:idx_user_provider,priority:1"`
	Provider   string    `gorm:"column:provider;type:varchar(32);not null;index:idx_user_provider,priority:2;uniqueIndex:idx_provider_app_external,priority:1"`
	ExternalID string    `gorm:"column:external_id;type:varchar(128);not null;uniqueIndex:idx_provider_app_external,priority:3"`
	AppID      *string   `gorm:"column:app_id;type:varchar(64);uniqueIndex:idx_provider_app_external,priority:2"`
	Status     int8      `gorm:"column:status;type:tinyint;not null;default:1"`
}

// WeChatAccountPO 微信账号持久化对象。
type WeChatAccountPO struct {
	base.AuditFields
	AccountID idutil.ID `gorm:"column:account_id;type:bigint unsigned;not null;uniqueIndex"`
	AppID     string    `gorm:"column:app_id;type:varchar(64);not null;index:app_open,priority:1"`
	OpenID    string    `gorm:"column:open_id;type:varchar(128);not null;index:app_open,priority:2"`
	UnionID   *string   `gorm:"column:union_id;type:varchar(128)"`
	Nickname  *string   `gorm:"column:nickname;type:varchar(128)"`
	AvatarURL *string   `gorm:"column:avatar_url;type:varchar(256)"`
	Meta      []byte    `gorm:"column:meta;type:json"`
}

// OperationAccountPO 运营后台账号凭证持久化对象。
type OperationAccountPO struct {
	base.AuditFields
	AccountID      idutil.ID  `gorm:"column:account_id;type:bigint unsigned;not null;uniqueIndex"`
	Username       string     `gorm:"column:username;type:varchar(64);not null;uniqueIndex"`
	PasswordHash   []byte     `gorm:"column:password_hash;type:varbinary(255);not null"`
	Algo           string     `gorm:"column:algo;type:varchar(32);not null"`
	Params         []byte     `gorm:"column:params;type:varbinary(512)"`
	FailedAttempts int        `gorm:"column:failed_attempts;type:int;not null;default:0"`
	LockedUntil    *time.Time `gorm:"column:locked_until;type:datetime"`
	LastChangedAt  time.Time  `gorm:"column:last_changed_at;type:datetime;not null"`
}

// TableName 指定账号表名。
func (AccountPO) TableName() string {
	return "iam_auth_accounts"
}

// TableName 指定微信账号表名。
func (WeChatAccountPO) TableName() string {
	return "iam_auth_wechat_accounts"
}

// TableName 指定运营账号凭证表名。
func (OperationAccountPO) TableName() string {
	return "iam_auth_operation_accounts"
}

// BeforeCreate 在创建前设置信息。
func (p *AccountPO) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	p.ID = idutil.NewID(idutil.GetIntID())
	p.CreatedAt = now
	p.UpdatedAt = now
	p.CreatedBy = idutil.NewID(0)
	p.UpdatedBy = idutil.NewID(0)
	p.DeletedBy = idutil.NewID(0)
	p.Version = base.InitialVersion
	return nil
}

// BeforeUpdate 在更新前设置信息。
func (p *AccountPO) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	p.UpdatedBy = idutil.NewID(0)
	return nil
}

// BeforeCreate 在创建前设置信息。
func (p *WeChatAccountPO) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	p.ID = idutil.NewID(idutil.GetIntID())
	p.CreatedAt = now
	p.UpdatedAt = now
	p.CreatedBy = idutil.NewID(0)
	p.UpdatedBy = idutil.NewID(0)
	p.DeletedBy = idutil.NewID(0)
	p.Version = base.InitialVersion
	return nil
}

// BeforeUpdate 在更新前设置信息。
func (p *WeChatAccountPO) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	p.UpdatedBy = idutil.NewID(0)
	return nil
}

// BeforeCreate 在创建前设置信息。
func (p *OperationAccountPO) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	p.ID = idutil.NewID(idutil.GetIntID())
	p.CreatedAt = now
	p.UpdatedAt = now
	p.CreatedBy = idutil.NewID(0)
	p.UpdatedBy = idutil.NewID(0)
	p.DeletedBy = idutil.NewID(0)
	p.Version = base.InitialVersion
	if p.LastChangedAt.IsZero() {
		p.LastChangedAt = now
	}
	return nil
}

// BeforeUpdate 在更新前设置信息。
func (p *OperationAccountPO) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	p.UpdatedBy = idutil.NewID(0)
	return nil
}
