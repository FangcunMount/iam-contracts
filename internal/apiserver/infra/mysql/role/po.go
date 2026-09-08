package role

import (
	"time"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	base "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"gorm.io/gorm"
)

// RolePO 角色持久化对象
type RolePO struct {
	base.AuditFields
	ManagementProtection string `gorm:"column:management_protection;type:varchar(16);not null;default:standard"`
	Name                 string `gorm:"column:name;type:varchar(64);not null;uniqueIndex:uk_role_name"`
	DisplayName          string `gorm:"column:display_name;type:varchar(128)"`

	IsSystem    uint8  `gorm:"column:is_system;type:tinyint;not null;default:0;comment:系统内置角色标识"`
	Description string `gorm:"column:description;type:varchar(512)"`
}

// TableName 指定表名
func (RolePO) TableName() string {
	return "authz_roles"
}

// BeforeCreate 在创建前设置信息
func (p *RolePO) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	id := meta.FromUint64(idutil.GetIntID()) // 新生成的 ID 必定有效
	createdBy := base.UserIDOrZero(tx.Statement.Context)
	updatedBy := createdBy
	deletedBy := meta.FromUint64(0)
	p.ID = id
	p.CreatedAt = now
	p.UpdatedAt = now
	p.CreatedBy = createdBy
	p.UpdatedBy = updatedBy
	p.DeletedBy = deletedBy
	p.Version = base.InitialVersion
	return nil
}

// BeforeUpdate 在更新前设置信息
func (p *RolePO) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	updatedBy := base.UserIDOrZero(tx.Statement.Context)
	p.UpdatedBy = updatedBy
	return nil
}
