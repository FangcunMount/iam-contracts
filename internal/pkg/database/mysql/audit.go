package mysql

import (
	"time"

	"github.com/fangcun-mount/iam-contracts/pkg/util/idutil"
)

// InitialVersion 默认的乐观锁版本号起点。
const InitialVersion uint32 = 1

// Syncable aggregates the behaviour required by persistence entities so that
// repositories can propagate auditing metadata back to domain models.
type Syncable interface {
	GetID() idutil.ID
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	GetDeletedAt() time.Time
	GetCreatedBy() idutil.ID
	GetUpdatedBy() idutil.ID
	GetDeletedBy() idutil.ID
	GetVersion() uint32
	SetID(idutil.ID)
	SetCreatedAt(time.Time)
	SetUpdatedAt(time.Time)
	SetDeletedAt(time.Time)
	SetCreatedBy(idutil.ID)
	SetUpdatedBy(idutil.ID)
	SetDeletedBy(idutil.ID)
	SetVersion(uint32)
}

// AuditFields provides reusable columns for ID and audit timestamps.
type AuditFields struct {
	ID        idutil.ID `gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt time.Time `gorm:"column:deleted_at;index"`
	CreatedBy idutil.ID `gorm:"column:created_by;type:varchar(50)" json:"created_by"`
	UpdatedBy idutil.ID `gorm:"column:updated_by;type:varchar(50)" json:"updated_by"`
	DeletedBy idutil.ID `gorm:"column:deleted_by;type:varchar(50)" json:"deleted_by"`
	Version   uint32    `gorm:"column:version;type:int unsigned;not null;default:1;version" json:"version"`
}

func (a *AuditFields) GetID() idutil.ID {
	return a.ID
}

func (a *AuditFields) GetCreatedAt() time.Time {
	return a.CreatedAt
}

func (a *AuditFields) GetUpdatedAt() time.Time {
	return a.UpdatedAt
}

func (a *AuditFields) GetDeletedAt() time.Time {
	return a.DeletedAt
}

func (a *AuditFields) GetCreatedBy() idutil.ID {
	return a.CreatedBy
}

func (a *AuditFields) GetUpdatedBy() idutil.ID {
	return a.UpdatedBy
}

func (a *AuditFields) GetDeletedBy() idutil.ID {
	return a.DeletedBy
}

func (a *AuditFields) GetVersion() uint32 {
	return a.Version
}

func (a *AuditFields) SetID(id idutil.ID) {
	a.ID = id
}

func (a *AuditFields) SetCreatedAt(t time.Time) {
	a.CreatedAt = t
}

func (a *AuditFields) SetUpdatedAt(t time.Time) {
	a.UpdatedAt = t
}

func (a *AuditFields) SetDeletedAt(t time.Time) {
	a.DeletedAt = t
}

func (a *AuditFields) SetCreatedBy(id idutil.ID) {
	a.CreatedBy = id
}

func (a *AuditFields) SetUpdatedBy(id idutil.ID) {
	a.UpdatedBy = id
}

func (a *AuditFields) SetDeletedBy(id idutil.ID) {
	a.DeletedBy = id
}

func (a *AuditFields) SetVersion(v uint32) {
	a.Version = v
}
