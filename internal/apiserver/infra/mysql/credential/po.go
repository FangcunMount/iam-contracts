package credential

import (
	"time"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	base "github.com/FangcunMount/iam-contracts/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"gorm.io/gorm"
)

// PO 持久化对象，对应凭据表。
type PO struct {
	base.AuditFields
	AccountID meta.ID `gorm:"column:account_id;type:bigint unsigned;not null;index:idx_account_type,priority:1"`
	Type      string  `gorm:"column:type;type:varchar(32);not null;index:idx_account_type,priority:2"`

	// 外部身份三元组
	IDP           *string `gorm:"column:idp;type:varchar(32)"`                                      // wechat/wecom/phone
	IDPIdentifier string  `gorm:"column:idp_identifier;type:varchar(255);index:idx_idp_identifier"` // unionid/openid@appid/userid/phone
	AppID         *string `gorm:"column:app_id;type:varchar(64)"`                                   // appid/corp_id

	// 凭据材料（password 专用）
	Material []byte  `gorm:"column:material;type:varbinary(512)"` // PHC hash
	Algo     *string `gorm:"column:algo;type:varchar(32)"`        // argon2id/bcrypt
	Params   []byte  `gorm:"column:params;type:json"`             // 扩展参数

	// 状态管理
	Status         int8       `gorm:"column:status;type:tinyint;not null;default:1"`
	FailedAttempts int        `gorm:"column:failed_attempts;type:int;not null;default:0"`
	LockedUntil    *time.Time `gorm:"column:locked_until;type:datetime"`
	LastSuccessAt  *time.Time `gorm:"column:last_success_at;type:datetime"`
	LastFailureAt  *time.Time `gorm:"column:last_failure_at;type:datetime"`

	Rev int64 `gorm:"column:rev;type:bigint;not null;default:0"` // 乐观锁版本号
}

// TableName 指定凭据表名。
func (PO) TableName() string {
	return "iam_auth_credentials"
}

// BeforeCreate 在创建前设置信息。
func (p *PO) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	p.ID = meta.NewID(idutil.GetIntID())
	p.CreatedAt = now
	p.UpdatedAt = now
	p.CreatedBy = meta.NewID(0)
	p.UpdatedBy = meta.NewID(0)
	p.DeletedBy = meta.NewID(0)
	p.Version = base.InitialVersion
	p.Rev = 0
	return nil
}

// BeforeUpdate 在更新前设置信息。
func (p *PO) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	p.UpdatedBy = meta.NewID(0)
	p.Rev++ // 乐观锁版本递增
	return nil
}
