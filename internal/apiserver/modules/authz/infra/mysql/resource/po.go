package resource

import (
	"time"

	base "github.com/fangcun-mount/iam-contracts/internal/pkg/database/mysql"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
	"gorm.io/gorm"
)

// ResourcePO 资源持久化对象
type ResourcePO struct {
	base.AuditFields
	Key         string `gorm:"column:key;type:varchar(128);not null;uniqueIndex"`
	DisplayName string `gorm:"column:display_name;type:varchar(128)"`
	AppName     string `gorm:"column:app_name;type:varchar(32);index"`
	Domain      string `gorm:"column:domain;type:varchar(32);index"`
	Type        string `gorm:"column:type;type:varchar(32);index"`
	Actions     string `gorm:"column:actions;type:text"` // JSON array string
	Description string `gorm:"column:description;type:varchar(512)"`
}

// TableName 指定表名
func (ResourcePO) TableName() string {
	return "authz_resources"
}

// BeforeCreate 在创建前设置信息
func (p *ResourcePO) BeforeCreate(tx *gorm.DB) error {
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
func (p *ResourcePO) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	p.UpdatedBy = idutil.NewID(0)
	return nil
}
