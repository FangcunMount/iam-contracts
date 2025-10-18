package policy

import (
	"time"

	base "github.com/fangcun-mount/iam-contracts/internal/pkg/database/mysql"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
	"gorm.io/gorm"
)

// PolicyVersionPO 策略版本持久化对象
type PolicyVersionPO struct {
	base.AuditFields
	TenantID  string `gorm:"column:tenant_id;type:varchar(64);not null;uniqueIndex"`
	Version   int64  `gorm:"column:version;type:bigint;not null"`
	ChangedBy string `gorm:"column:changed_by;type:varchar(64)"`
	Reason    string `gorm:"column:reason;type:varchar(512)"`
}

// TableName 指定表名
func (PolicyVersionPO) TableName() string {
	return "authz_policy_versions"
}

// BeforeCreate 在创建前设置信息
func (p *PolicyVersionPO) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	p.ID = idutil.NewID(idutil.GetIntID())
	p.CreatedAt = now
	p.UpdatedAt = now
	p.CreatedBy = idutil.NewID(0)
	p.UpdatedBy = idutil.NewID(0)
	p.DeletedBy = idutil.NewID(0)
	// Note: Version field is policy version, not audit version
	return nil
}

// BeforeUpdate 在更新前设置信息
func (p *PolicyVersionPO) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	p.UpdatedBy = idutil.NewID(0)
	return nil
}
