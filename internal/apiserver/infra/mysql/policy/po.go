package policy

import (
	"time"

	base "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"gorm.io/gorm"
)

// PolicyVersionPO 策略版本持久化对象
type PolicyVersionPO struct {
	base.AuditFields

	PolicyVersion int64  `gorm:"column:policy_version;type:bigint;not null"`
	ChangedBy     string `gorm:"column:changed_by;type:varchar(64)"`
	Reason        string `gorm:"column:reason;type:varchar(512)"`
}

// TableName 指定表名
func (PolicyVersionPO) TableName() string {
	return "authz_policy_versions"
}

// BeforeCreate 在创建前设置信息
func (p *PolicyVersionPO) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	id := meta.ID(1)
	createdBy := base.UserIDOrZero(tx.Statement.Context)
	updatedBy := createdBy
	deletedBy := meta.FromUint64(0)
	p.ID = id
	p.CreatedAt = now
	p.UpdatedAt = now
	p.CreatedBy = createdBy
	p.UpdatedBy = updatedBy
	p.DeletedBy = deletedBy
	p.Version = base.InitialVersion // AuditFields 的版本字段
	return nil
}

// BeforeUpdate 在更新前设置信息
func (p *PolicyVersionPO) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	updatedBy := base.UserIDOrZero(tx.Statement.Context)
	p.UpdatedBy = updatedBy
	return nil
}
