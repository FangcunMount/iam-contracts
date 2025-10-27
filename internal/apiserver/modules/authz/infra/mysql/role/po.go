package role

import (
	"time"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	base "github.com/fangcun-mount/iam-contracts/internal/pkg/database/mysql"
	"gorm.io/gorm"
)

// RolePO 角色持久化对象
type RolePO struct {
	base.AuditFields
	Name        string `gorm:"column:name;type:varchar(64);not null;uniqueIndex:uk_tenant_name,priority:2"`
	DisplayName string `gorm:"column:display_name;type:varchar(128)"`
	TenantID    string `gorm:"column:tenant_id;type:varchar(64);not null;uniqueIndex:uk_tenant_name,priority:1;index"`
	Description string `gorm:"column:description;type:varchar(512)"`
}

// TableName 指定表名
func (RolePO) TableName() string {
	return "authz_roles"
}

// BeforeCreate 在创建前设置信息
func (p *RolePO) BeforeCreate(tx *gorm.DB) error {
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

// BeforeUpdate 在更新前设置信息
func (p *RolePO) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	p.UpdatedBy = idutil.NewID(0)
	return nil
}
