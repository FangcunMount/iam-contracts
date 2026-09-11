// Historical schema 32 preparation fixture; never linked into a service.
package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/constraint"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"gorm.io/gorm"
)

// retirementPreparation 仅供离线迁移使用；调用期间必须停止全部授权写入。
type retirementPreparation struct {
	ID             uint8     `gorm:"primaryKey"`
	SourceHash     string    `gorm:"size:64;not null"`
	PreparedHash   string    `gorm:"size:64;not null"`
	InitialVersion int64     `gorm:"not null"`
	PreparedAt     time.Time `gorm:"not null"`
}

func (retirementPreparation) TableName() string { return "iam_authorization_retirement_preparation" }

// PrepareTenantRetirement 将已预检的历史事实转换为新模型。预检失败不修改授权数据。
// 指纹必须来自当前备份对应的预检；DDL 切换由下一顺序迁移执行。
func PrepareTenantRetirement(ctx context.Context, db *gorm.DB, fingerprint string) (*TenantRetirementReport, error) {
	if len(fingerprint) != 64 {
		return nil, fmt.Errorf("必须提供预检指纹")
	}
	before, err := AnalyzeTenantRetirement(ctx, db)
	if err != nil {
		return nil, err
	}
	if !before.Ready {
		return before, fmt.Errorf("迁移预检未通过")
	}
	if db.Migrator().HasTable(&retirementPreparation{}) {
		var marker retirementPreparation
		if err := db.First(&marker, 1).Error; err == nil && marker.SourceHash == fingerprint && marker.PreparedHash == before.Fingerprint {
			return before, nil
		}
	}
	if before.Fingerprint != fingerprint {
		return before, fmt.Errorf("预检指纹已变化，请重新备份并预检")
	}
	if err := db.AutoMigrate(&retirementPreparation{}); err != nil {
		return nil, err
	}
	if err := installRetirementInvalidationTriggers(db); err != nil {
		return nil, err
	}
	var after *TenantRetirementReport
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		state, err := loadRetirementState(ctx, tx, true)
		if err != nil {
			return err
		}
		report := analyzeRetirement(state)
		if err := checkRetirementCollation(tx, report); err != nil {
			return err
		}
		if !report.Ready || report.Fingerprint != fingerprint {
			return fmt.Errorf("授权事实已变化，拒绝迁移")
		}
		for _, r := range report.Roles {
			if err := tx.Table("authz_roles").Where("id = ?", r.ID).Updates(map[string]any{"name": r.After, "management_protection": r.Protection}).Error; err != nil {
				return err
			}
		}
		for id, key := range report.grantKeys {
			if err := tx.Table("authz_permission_grants").Where("id = ?", id).Update("grant_key", key).Error; err != nil {
				return err
			}
		}
		if err := prepareProfileCapabilities(tx, state); err != nil {
			return err
		}
		prepared, err := loadRetirementState(ctx, tx, false)
		if err != nil {
			return err
		}
		after = analyzeRetirement(prepared)
		if !after.Ready {
			return fmt.Errorf("转换后授权事实未通过校验")
		}
		marker := retirementPreparation{ID: 1, SourceHash: fingerprint, PreparedHash: after.Fingerprint, InitialVersion: report.InitialPolicyVersion, PreparedAt: time.Now().UTC()}
		return tx.Save(&marker).Error
	})
	return after, err
}

func prepareProfileCapabilities(tx *gorm.DB, state retirementState) error {
	const profileKey = "iam:identity:collection:profiles"
	var profileID uint64
	for _, r := range state.Resources {
		if r.DeletedAt != nil {
			continue
		}
		var additions []string
		switch r.Key {
		case profileKey:
			profileID = r.ID.Uint64()
			additions = []string{"list_all", "search_by_mobile_all"}
		case "iam:authz:collection:roles":
			additions = []string{"manage_protected"}
		}
		for _, action := range additions {
			if err := tx.Exec("UPDATE authz_resources SET actions = JSON_ARRAY_APPEND(actions, '$', ?) WHERE id = ? AND JSON_CONTAINS(actions, JSON_QUOTE(?)) = 0", action, r.ID, action).Error; err != nil {
				return err
			}
		}
	}
	for _, g := range state.Grants {
		if g.LegacyDomain != "platform" || g.DeletedAt != nil || g.RevokedAt != nil || !resource.Key(g.ResourcePattern).Covers(resource.Key(profileKey)) {
			continue
		}
		action := ""
		switch g.Action {
		case "list":
			action = "list_all"
		case "search_by_mobile":
			action = "search_by_mobile_all"
		}
		if action == "" {
			continue
		} // 动作通配符已经覆盖新能力。
		if profileID == 0 {
			return fmt.Errorf("缺少档案资源目录，不能迁移全量能力")
		}
		c, err := constraint.ParseJSON([]byte(g.ConstraintSet))
		if err != nil {
			return err
		}
		grant, err := permissiongrant.Restore(meta.ID(g.RoleID), resource.NewResourceID(profileID), profileKey, action, g.GrantedBy, permissiongrant.RestoreOptions{})
		if err != nil {
			return err
		}
		var count int64
		if err := tx.Table("authz_permission_grants").Where("grant_key = ? AND deleted_at IS NULL AND revoked_at IS NULL", grant.GrantKey).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		canonical, err := c.CanonicalJSON()
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		values := map[string]any{"id": idutil.GetIntID(), "tenant_id": "platform", "role_id": g.RoleID, "resource_id": profileID, "resource_pattern": profileKey, "action": action, "constraint_set": string(canonical), "grant_key": grant.GrantKey, "granted_by": g.GrantedBy, "granted_at": g.GrantedAt, "created_at": now, "updated_at": now, "version": 1}
		if err := tx.Table("authz_permission_grants").Create(values).Error; err != nil {
			return err
		}
	}
	return nil
}

// 任何后续授权写入都会清除准备凭据，阻止过期预检进入 DDL 切换。
func installRetirementInvalidationTriggers(db *gorm.DB) error {
	if db.Dialector.Name() != "mysql" {
		return fmt.Errorf("迁移准备只支持 MySQL 8")
	}
	for _, table := range []string{"authz_roles", "authz_assignments", "authz_role_inheritances", "authz_permission_grants", "authz_resources", "authz_policy_versions"} {
		for _, operation := range []string{"INSERT", "UPDATE", "DELETE"} {
			name := "iam_retire_" + table + "_" + operation
			var count int64
			if err := db.Raw("SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE() AND TRIGGER_NAME=?", name).Scan(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				if err := db.Exec("CREATE TRIGGER `" + name + "` AFTER " + operation + " ON `" + table + "` FOR EACH ROW DELETE FROM iam_authorization_retirement_preparation WHERE id=1").Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}
