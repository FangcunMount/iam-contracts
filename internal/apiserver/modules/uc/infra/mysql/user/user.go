package user

import (
	"time"

	base "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/infra/mysql"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
	"gorm.io/gorm"
)

// UserPO 用户持久化对象
// 对应数据库表结构
type UserPO struct {
	base.AuditFields
	Name   string `gorm:"column:name;type:varchar(64);not null;comment:用户名称"`
	Phone  string `gorm:"column:phone;type:varchar(20);uniqueIndex;not null;comment:手机号"`
	Email  string `gorm:"column:email;type:varchar(100);uniqueIndex;not null;comment:邮箱"`
	IDCard string `gorm:"column:id_card;type:varchar(20);uniqueIndex;not null;comment:身份证号"`
	Status uint8  `gorm:"column:status;type:int;not null;default:1;comment:用户状态"`
}

// TableName 指定表名
func (UserPO) TableName() string {
	return "users"
}

// BeforeCreate 在创建前设置信息
func (p *UserPO) BeforeCreate(tx *gorm.DB) error {
	p.ID = idutil.NewID(idutil.GetIntID())
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	p.CreatedBy = idutil.NewID(0)
	p.UpdatedBy = idutil.NewID(0)
	p.DeletedBy = idutil.NewID(0)

	return nil
}

// BeforeUpdate 在更新前设置信息
func (p *UserPO) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	p.UpdatedBy = idutil.NewID(0)

	return nil
}
