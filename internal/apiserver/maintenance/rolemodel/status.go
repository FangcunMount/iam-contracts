package rolemodel

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"gorm.io/gorm"
)

// StatusReport separates historical completion from current authorization facts.
// Changed sections contain names only; personnel evidence stays in the receipt.
type StatusReport struct {
	State                  string   `json:"state"`
	NextAction             string   `json:"next_action"`
	MigrationID            string   `json:"migration_id"`
	Fingerprint            string   `json:"fingerprint"`
	AfterHash              string   `json:"after_hash"`
	CurrentHash            string   `json:"current_hash"`
	ChangedSections        []string `json:"changed_sections"`
	InheritanceTableExists bool     `json:"inheritance_table_exists"`
	ArchiveExists          bool     `json:"archive_exists"`
}

func Status(ctx context.Context, db *gorm.DB) (StatusReport, error) {
	var report StatusReport
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		report, err = inspectStatus(ctx, tx)
		return err
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return report, err
}

func inspectStatus(ctx context.Context, db *gorm.DB) (StatusReport, error) {
	r := StatusReport{State: "invalid", NextAction: "inspect", MigrationID: MigrationID, ChangedSections: []string{}}
	if !db.Migrator().HasTable(&Receipt{}) {
		r.State, r.NextAction = "pending", "preflight"
		return r, nil
	}
	var receipt Receipt
	err := db.WithContext(ctx).First(&receipt, "migration_id = ?", MigrationID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		r.State, r.NextAction = "pending", "preflight"
		return r, nil
	}
	if err != nil {
		return r, err
	}
	r.Fingerprint, r.AfterHash = receipt.Fingerprint, receipt.AfterHash
	var before, after State
	var plan Plan
	if digest, err := hex.DecodeString(receipt.Fingerprint); err != nil || len(digest) != 32 {
		return r, fmt.Errorf("invalid receipt fingerprint")
	}
	if json.Unmarshal([]byte(receipt.BeforeJSON), &before) != nil || json.Unmarshal([]byte(receipt.AfterJSON), &after) != nil || json.Unmarshal([]byte(receipt.PlanJSON), &plan) != nil || after.Hash() != receipt.AfterHash || plan.Fingerprint != receipt.Fingerprint || plan.MigrationID != MigrationID || plan.Validate() != nil {
		return r, fmt.Errorf("invalid migration receipt")
	}
	if len(before.Roles) == 0 || len(after.Roles) == 0 {
		return r, fmt.Errorf("missing receipt facts")
	}
	if err := verifyPlan(after, plan); err != nil {
		return r, fmt.Errorf("invalid receipt after-image: %w", err)
	}
	if receipt.Status == "rolled_back" {
		r.State, r.NextAction = "rolled_back", "review_new_migration"
		return r, nil
	}
	if receipt.Status != "applied" {
		return r, fmt.Errorf("invalid migration status")
	}
	current, err := loadStateForVerification(ctx, db)
	if err != nil {
		return r, err
	}
	r.InheritanceTableExists = db.Migrator().HasTable(&LegacyEdge{})
	if db.Migrator().HasTable(&InheritanceArchive{}) {
		var a InheritanceArchive
		result := db.WithContext(ctx).Where("migration_id = ?", MigrationID).Limit(1).Find(&a)
		if result.Error != nil {
			return r, result.Error
		}
		r.ArchiveExists = result.RowsAffected != 0
		if result.RowsAffected != 0 && (archiveChecksum(a.SchemaSQL, a.RowsJSON) != a.Checksum || a.AfterHash != receipt.AfterHash || a.RowsJSON != encode(after.Edges)) {
			return r, fmt.Errorf("invalid inheritance archive")
		}
	}
	r.CurrentHash = current.Hash()
	for _, section := range []struct {
		name            string
		before, current any
	}{
		{"roles", after.Roles, current.Roles}, {"assignments", after.Assignments, current.Assignments},
		{"grants", after.Grants, current.Grants}, {"resources", after.Resources, current.Resources},
		{"inheritance_history", after.Edges, current.Edges}, {"policy_version", after.PolicyVersion, current.PolicyVersion},
	} {
		if !reflect.DeepEqual(section.before, section.current) {
			r.ChangedSections = append(r.ChangedSections, section.name)
		}
	}
	r.State, r.NextAction = "applied_unchanged", "none"
	if r.InheritanceTableExists {
		r.NextAction = "retire_inheritance"
	}
	if !r.ArchiveExists {
		r.NextAction = "archive_inheritance"
	}
	if r.CurrentHash != r.AfterHash {
		r.State, r.NextAction = "applied_drifted", "reconcile"
	}
	return r, nil
}
