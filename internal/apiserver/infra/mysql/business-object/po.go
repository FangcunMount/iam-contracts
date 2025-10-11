package business_object

import (
	"time"

	base "github.com/fangcun-mount/iam-contracts/internal/apiserver/infra/mysql"
)

// TesteePO 被测者持久化对象
type TesteePO struct {
	base.AuditFields
	UserID   uint64     `gorm:"column:user_id;not null;uniqueIndex:uidx_testees_user_id"`
	Name     string     `gorm:"column:name;type:varchar(64);not null"`
	Sex      uint8      `gorm:"column:sex;type:tinyint unsigned;not null;default:0"`
	Birthday *time.Time `gorm:"column:birthday;type:date"`
}

// TableName 指定表名
func (TesteePO) TableName() string {
	return "biz_testees"
}

// AuditorPO 审核员持久化对象
type AuditorPO struct {
	base.AuditFields
	UserID     uint64     `gorm:"column:user_id;not null;uniqueIndex:uidx_auditors_user_id"`
	Name       string     `gorm:"column:name;type:varchar(64);not null"`
	EmployeeID string     `gorm:"column:employee_id;type:varchar(64);not null;uniqueIndex"`
	Department string     `gorm:"column:department;type:varchar(128)"`
	Position   string     `gorm:"column:position;type:varchar(128)"`
	Status     uint8      `gorm:"column:status;type:tinyint unsigned;not null;default:1"`
	HiredAt    *time.Time `gorm:"column:hired_at"`
}

// TableName 指定表名
func (AuditorPO) TableName() string {
	return "biz_auditors"
}
