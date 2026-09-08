package maintenance

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"gorm.io/gorm"
)

// retirementAudit 保存离线治理前的完整行和处置说明；数据库备份才是恢复来源。
// 本表不会被运行时或迁移清除。
type retirementAudit struct {
	Batch        string    `gorm:"primaryKey;size:64"`
	SourceTable  string    `gorm:"primaryKey;size:64"`
	SourceID     uint64    `gorm:"primaryKey"`
	BackupSHA256 string    `gorm:"size:64;not null"`
	OriginalJSON string    `gorm:"type:longtext;not null"`
	Disposition  string    `gorm:"size:64;not null"`
	ArchivedAt   time.Time `gorm:"not null"`
}

func (retirementAudit) TableName() string { return "iam_authorization_retirement_audit" }

// ArchiveApprovedRetirementIssues 仅用于已停止流量、写入及消费者的切换窗口。
// 只处理逐项批准且仍与预检一致的历史孤儿和普通角色敏感授权。
// 不修复引用、不提升保护等级；全局版本由随后迁移生成，切换前不得恢复旧实例。
func ArchiveApprovedRetirementIssues(ctx context.Context, db *gorm.DB, fingerprint, backupSHA256 string, approvedIDs []string) (*TenantRetirementReport, error) {
	if bytes, err := hex.DecodeString(backupSHA256); err != nil || len(bytes) != 32 {
		return nil, fmt.Errorf("必须提供完整数据库备份的 SHA256")
	}
	before, err := AnalyzeTenantRetirement(ctx, db)
	if err != nil {
		return nil, err
	}
	if before.Fingerprint != fingerprint || before.Ready {
		return before, fmt.Errorf("待治理预检已变化")
	}
	if err := validateApprovedIssues(before, approvedIDs); err != nil {
		return before, err
	}
	if err := db.AutoMigrate(&retirementAudit{}); err != nil {
		return before, err
	}
	var after *TenantRetirementReport
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		state, err := loadRetirementState(ctx, tx, true)
		if err != nil {
			return err
		}
		current := analyzeRetirement(state)
		if current.Fingerprint != fingerprint {
			return fmt.Errorf("授权事实已变化，拒绝治理")
		}
		if err := validateApprovedIssues(current, approvedIDs); err != nil {
			return err
		}
		type target struct {
			table       string
			id          meta.ID
			active      bool
			disposition string
		}
		targets := map[string]target{}
		for _, a := range state.Assignments {
			if slices.Contains(approvedIDs, a.ID.String()) {
				targets[a.ID.String()] = target{"authz_assignments", a.ID, a.DeletedAt == nil, "archived_inactive_orphan"}
			}
		}
		for _, e := range state.Inheritances {
			if slices.Contains(approvedIDs, e.ID.String()) {
				targets[e.ID.String()] = target{"authz_role_inheritances", e.ID, e.DeletedAt == nil && e.RevokedAt == nil, "archived_inactive_orphan"}
			}
		}
		for _, g := range state.Grants {
			if slices.Contains(approvedIDs, g.ID.String()) {
				targets[g.ID.String()] = target{"authz_permission_grants", g.ID, g.DeletedAt == nil && g.RevokedAt == nil, "archived_inactive_orphan"}
			}
		}
		for _, issue := range current.Issues {
			t, ok := targets[issue.ID]
			if !ok {
				return fmt.Errorf("治理目标不是可归档的授权事实")
			}
			if issue.Kind != "sensitive_standard_grant" && t.active {
				return fmt.Errorf("不得归档活跃孤儿事实")
			}
			if issue.Kind == "sensitive_standard_grant" {
				t.disposition = "revoked_and_archived_sensitive_grant"
				targets[issue.ID] = t
			}
		}
		now := time.Now().UTC()
		for _, id := range approvedIDs {
			t := targets[id]
			var row map[string]any
			if err := tx.Table(t.table).Where("id = ?", t.id).Take(&row).Error; err != nil {
				return err
			}
			encoded, err := json.Marshal(row)
			if err != nil {
				return err
			}
			audit := retirementAudit{Batch: fingerprint, SourceTable: t.table, SourceID: t.id.Uint64(), BackupSHA256: backupSHA256, OriginalJSON: string(encoded), Disposition: t.disposition, ArchivedAt: now}
			if err := tx.Create(&audit).Error; err != nil {
				return err
			}
			if t.active {
				if err := tx.Table(t.table).Where("id = ?", t.id).Update("revoked_at", now).Error; err != nil {
					return err
				}
			}
			result := tx.Exec("DELETE FROM "+t.table+" WHERE id = ?", t.id)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return fmt.Errorf("治理目标数量变化")
			}
		}
		remaining, err := loadRetirementState(ctx, tx, false)
		if err != nil {
			return err
		}
		after = analyzeRetirement(remaining)
		if err := checkRetirementCollation(tx, after); err != nil {
			return err
		}
		if !after.Ready {
			return fmt.Errorf("治理后仍存在阻断，整批回滚")
		}
		return nil
	})
	return after, err
}

func validateApprovedIssues(report *TenantRetirementReport, approved []string) error {
	if len(approved) == 0 {
		return fmt.Errorf("必须逐项指定已批准的记录 ID")
	}
	expected := map[string]bool{}
	for _, issue := range report.Issues {
		switch issue.Kind {
		case "invalid_role_reference", "invalid_resource_reference", "sensitive_standard_grant":
		default:
			return fmt.Errorf("存在本次治理不支持的预检问题")
		}
		expected[issue.ID] = true
	}
	if len(expected) != len(approved) {
		return fmt.Errorf("批准的记录集合与当前预检不一致")
	}
	seen := map[string]bool{}
	for _, id := range approved {
		if !expected[id] || seen[id] {
			return fmt.Errorf("批准的记录集合与当前预检不一致")
		}
		seen[id] = true
	}
	return nil
}
