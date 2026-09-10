package rolemodel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"time"
)

// InheritanceArchive is an exact history image plus the original MySQL DDL.
// It is retained after the live table is removed and exported with the receipt.
type InheritanceArchive struct {
	MigrationID string    `gorm:"primaryKey;size:64" json:"migration_id"`
	SchemaSQL   string    `gorm:"type:longtext" json:"schema_sql"`
	RowsJSON    string    `gorm:"type:longtext" json:"rows_json"`
	Checksum    string    `gorm:"size:64" json:"checksum"`
	AfterHash   string    `gorm:"size:64" json:"after_hash"`
	CreatedAt   time.Time `json:"created_at"`
}

func (InheritanceArchive) TableName() string { return "iam_role_inheritance_archives" }
func archiveChecksum(schema, rows string) string {
	sum := sha256.Sum256([]byte(schema + "\x00" + rows))
	return hex.EncodeToString(sum[:])
}

// ArchiveInheritance prepares the guard required by the final DDL migration.
// It never drops the table; keep writers stopped until that migration completes.
func ArchiveInheritance(ctx context.Context, db *gorm.DB, fingerprint string, stopped bool) (*InheritanceArchive, error) {
	if !stopped {
		return nil, fmt.Errorf("inheritance archival requires stopped writers")
	}
	receipt, err := Verify(ctx, db)
	if err != nil {
		return nil, err
	}
	if receipt.Fingerprint != fingerprint {
		return nil, fmt.Errorf("archive fingerprint mismatch")
	}
	if db.Dialector.Name() != "mysql" {
		return nil, fmt.Errorf("inheritance DDL archival requires MySQL")
	}
	var schema struct {
		Table       string
		CreateTable string `gorm:"column:Create Table"`
	}
	if err = db.WithContext(ctx).Raw("SHOW CREATE TABLE authz_role_inheritances").Scan(&schema).Error; err != nil {
		return nil, err
	}
	if schema.CreateTable == "" {
		return nil, fmt.Errorf("original inheritance DDL is missing")
	}
	var after State
	if err = json.Unmarshal([]byte(receipt.AfterJSON), &after); err != nil {
		return nil, err
	}
	rows := encode(after.Edges)
	archive := InheritanceArchive{MigrationID: MigrationID, SchemaSQL: schema.CreateTable, RowsJSON: rows, Checksum: archiveChecksum(schema.CreateTable, rows), AfterHash: receipt.AfterHash, CreatedAt: time.Now().UTC()}
	if err = db.AutoMigrate(&InheritanceArchive{}); err != nil {
		return nil, err
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPolicy(tx); err != nil {
			return err
		}
		if _, err := Verify(ctx, tx); err != nil {
			return err
		}
		var existing InheritanceArchive
		result := tx.Where("migration_id = ?", MigrationID).Limit(1).Find(&existing)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 0 {
			if existing.Checksum != archive.Checksum || existing.AfterHash != archive.AfterHash {
				return fmt.Errorf("inheritance archive conflicts")
			}
			archive = existing
			return nil
		}
		return tx.Create(&archive).Error
	})
	return &archive, err
}

// loadStateForVerification reconstructs only the removed historical table from
// its verified archive. Live roles, grants and assignments are always re-read.
func loadStateForVerification(ctx context.Context, db *gorm.DB) (State, error) {
	s, err := LoadState(ctx, db)
	if err != nil || db.Migrator().HasTable(&LegacyEdge{}) {
		return s, err
	}
	var a InheritanceArchive
	if err = db.WithContext(ctx).First(&a, "migration_id = ?", MigrationID).Error; err != nil {
		return s, err
	}
	if archiveChecksum(a.SchemaSQL, a.RowsJSON) != a.Checksum {
		return s, fmt.Errorf("inheritance archive checksum mismatch")
	}
	if err = json.Unmarshal([]byte(a.RowsJSON), &s.Edges); err != nil {
		return s, err
	}
	return s, nil
}
