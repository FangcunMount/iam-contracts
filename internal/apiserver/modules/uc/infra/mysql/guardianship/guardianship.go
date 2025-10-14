package guardianship

import (
	"time"

	base "github.com/fangcun-mount/iam-contracts/internal/pkg/database/mysql"
	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
	"gorm.io/gorm"
)

// GuardianshipPO 监护关系持久化对象
// 对应数据库表结构
type GuardianshipPO struct {
	base.AuditFields
	UserID        idutil.ID  `gorm:"column:user_id;type:bigint;not null;index;comment:监护人ID"`
	ChildID       idutil.ID  `gorm:"column:child_id;type:bigint;not null;index;comment:儿童ID"`
	Relation      string     `gorm:"column:relation;type:varchar(16);not null;comment:监护关系"`
	EstablishedAt time.Time  `gorm:"column:established_at;type:datetime;not null;comment:建立时间"`
	RevokedAt     *time.Time `gorm:"column:revoked_at;type:datetime;comment:撤销时间"`
}

// TableName 指定表名
func (GuardianshipPO) TableName() string {
	return "guardianships"
}

// BeforeCreate 在创建前设置信息
func (p *GuardianshipPO) BeforeCreate(tx *gorm.DB) error {
	p.ID = idutil.NewID(idutil.GetIntID())
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	p.CreatedBy = idutil.NewID(0)
	p.UpdatedBy = idutil.NewID(0)
	p.DeletedBy = idutil.NewID(0)

	if p.EstablishedAt.IsZero() {
		p.EstablishedAt = now
	}

	return nil
}

// BeforeUpdate 在更新前设置信息
func (p *GuardianshipPO) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	p.UpdatedBy = idutil.NewID(0)

	return nil
}
