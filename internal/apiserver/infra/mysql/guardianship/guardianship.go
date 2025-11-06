package guardianship

import (
	"time"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	base "github.com/FangcunMount/iam-contracts/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"gorm.io/gorm"
)

// GuardianshipPO 监护关系持久化对象
// 对应数据库表结构
type GuardianshipPO struct {
	base.AuditFields
	UserID        meta.ID    `gorm:"column:user_id;type:bigint unsigned;not null;index:idx_user_child_ref,priority:1;comment:监护人ID"`
	ChildID       meta.ID    `gorm:"column:child_id;type:bigint unsigned;not null;index:idx_user_child_ref,priority:2;comment:儿童ID"`
	Relation      string     `gorm:"column:relation;type:varchar(16);not null;comment:监护关系"`
	EstablishedAt time.Time  `gorm:"column:established_at;type:datetime;not null;comment:建立时间"`
	RevokedAt     *time.Time `gorm:"column:revoked_at;type:datetime;comment:撤销时间"`
}

// TableName 指定表名
func (GuardianshipPO) TableName() string {
	return "iam_guardianships"
}

// BeforeCreate 在创建前设置信息
func (p *GuardianshipPO) BeforeCreate(tx *gorm.DB) error {
	p.ID = meta.NewID(idutil.GetIntID())
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	p.CreatedBy = meta.NewID(0)
	p.UpdatedBy = meta.NewID(0)
	p.DeletedBy = meta.NewID(0)
	p.Version = base.InitialVersion

	if p.EstablishedAt.IsZero() {
		p.EstablishedAt = now
	}

	return nil
}

// BeforeUpdate 在更新前设置信息
func (p *GuardianshipPO) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	p.UpdatedBy = meta.NewID(0)

	return nil
}
