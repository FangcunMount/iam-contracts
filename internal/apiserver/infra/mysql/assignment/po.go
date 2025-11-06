package assignment

import (
	"time"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	base "github.com/FangcunMount/iam-contracts/internal/pkg/database/mysql"
	"gorm.io/gorm"
)

// AssignmentPO 赋权持久化对象
type AssignmentPO struct {
	base.AuditFields
	SubjectType string    `gorm:"column:subject_type;type:varchar(16);not null;index:idx_subject,priority:1"`
	SubjectID   string    `gorm:"column:subject_id;type:varchar(64);not null;index:idx_subject,priority:2"`
	RoleID      uint64    `gorm:"column:role_id;type:bigint unsigned;not null;index"`
	TenantID    string    `gorm:"column:tenant_id;type:varchar(64);not null;index"`
	GrantedBy   string    `gorm:"column:granted_by;type:varchar(64)"`
	GrantedAt   time.Time `gorm:"column:granted_at;type:datetime"`
}

// TableName 指定表名
func (AssignmentPO) TableName() string {
	return "iam_authz_assignments"
}

// BeforeCreate 在创建前设置信息
func (p *AssignmentPO) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	p.ID = idutil.NewID(idutil.GetIntID())
	p.CreatedAt = now
	p.UpdatedAt = now
	p.GrantedAt = now
	p.CreatedBy = idutil.NewID(0)
	p.UpdatedBy = idutil.NewID(0)
	p.DeletedBy = idutil.NewID(0)
	p.Version = base.InitialVersion
	return nil
}

// BeforeUpdate 在更新前设置信息
func (p *AssignmentPO) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	p.UpdatedBy = idutil.NewID(0)
	return nil
}
