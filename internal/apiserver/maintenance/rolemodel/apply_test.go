package rolemodel

import (
	"context"
	"encoding/json"
	"fmt"
	drivermysql "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"os"
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/constraint"
	grantdomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	resourcedomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	assignmentpo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/assignment"
	grantpo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/permissiongrant"
	policypo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/policy"
	resourcepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/resource"
	rolepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/role"
	"github.com/FangcunMount/iam/v5/pkg/event"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type recordingStager struct {
	fail     bool
	versions []int64
}

func (s *recordingStager) Stage(_ context.Context, es ...event.DomainEvent) error {
	if s.fail {
		return fmt.Errorf("outbox unavailable")
	}
	for range es {
		s.versions = append(s.versions, 1)
	}
	return nil
}
func migrationDB(t *testing.T) (*gorm.DB, *gorm.DB) {
	t.Helper()
	db := openMigrationTestDB(t, "iam")
	qs := openMigrationTestDB(t, "qs")
	var err error
	require.NoError(t, db.AutoMigrate(&rolepo.RolePO{}, &resourcepo.ResourcePO{}, &assignmentpo.AssignmentPO{}, &grantpo.GrantPO{}, &LegacyEdge{}, &policypo.PolicyVersionPO{}, &Receipt{}))
	require.NoError(t, db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, status INTEGER, deleted_at DATETIME)").Error)
	require.NoError(t, qs.Exec("CREATE TABLE staff (id INTEGER PRIMARY KEY, user_id INTEGER, org_id INTEGER, is_active BOOLEAN, deleted_at DATETIME)").Error)
	require.NoError(t, qs.Exec("CREATE TABLE clinician (id INTEGER PRIMARY KEY, operator_id INTEGER, org_id INTEGER, is_active BOOLEAN, deleted_at DATETIME)").Error)
	input := fixture()
	roles := map[string]rolepo.RolePO{}
	resources := map[string]resourcepo.ResourcePO{}
	for _, r := range input.Roles {
		p := rolepo.RolePO{Name: r.Name, DisplayName: r.Name, ManagementProtection: r.Protection}
		require.NoError(t, db.Create(&p).Error)
		roles[r.Name] = p
	}
	for _, r := range input.Resources {
		if r.Key == "*:*:*:*" || r.Key == "qs:*:*:*" {
			continue
		}
		p := resourcepo.ResourcePO{Key: r.Key, DisplayName: r.Key, Actions: encode(r.Actions), AttributeSchema: `{"version":1,"attributes":[]}`}
		if r.Key == Assessment {
			p.AttributeSchema = `{"version":1,"attributes":[{"key":"object.origin_type","type":"string","allowed_string_values":["adhoc","plan"]}]}`
		}
		require.NoError(t, db.Create(&p).Error)
		resources[r.Key] = p
	}
	for _, g := range input.Grants {
		c := constraint.Empty()
		if g.Permission.Origin != "" {
			c, err = constraint.New(constraint.Equal("object.origin_type", constraint.StringValue(g.Permission.Origin)))
			require.NoError(t, err)
		}
		rid := resourcedomain.NewResourceID(resources[g.Permission.Resource].ID.Uint64())
		var b grantdomain.Grant
		if rid.Uint64() == 0 {
			b, err = grantdomain.NewSystem(roles[g.Role].ID, rid, g.Permission.Resource, g.Permission.Action, c, "seed")
		} else {
			b, err = grantdomain.New(roles[g.Role].ID, rid, g.Permission.Resource, g.Permission.Action, c, "seed")
		}
		require.NoError(t, err)
		p, err := (grantpo.Mapper{}).ToPO(&b)
		require.NoError(t, err)
		require.NoError(t, db.Create(p).Error)
	}
	for i, e := range input.Edges {
		require.NoError(t, db.Create(&LegacyEdge{ID: uint64(i + 1), RoleID: roles[e.Role].ID.Uint64(), InheritedRoleID: roles[e.Parent].ID.Uint64(), Version: 1}).Error)
	}
	require.NoError(t, db.Create(&policypo.PolicyVersionPO{PolicyVersion: 11}).Error)
	require.NoError(t, db.Exec("INSERT INTO users(id,status) VALUES (1,1),(2,1)").Error)
	require.NoError(t, qs.Exec("INSERT INTO staff(id,user_id,org_id,is_active) VALUES (10,1,1,1),(20,2,1,1)").Error)
	require.NoError(t, qs.Exec("INSERT INTO clinician(id,operator_id,org_id,is_active) VALUES (100,10,1,1)").Error)
	for _, id := range []string{"1", "2"} {
		require.NoError(t, db.Create(&assignmentpo.AssignmentPO{SubjectType: "user", SubjectID: id, RoleID: roles["qs:staff"].ID.Uint64(), GrantedBy: "seed"}).Error)
	}
	return db, qs
}
func TestApplyIdempotencyAndRollbackPreserveFacts(t *testing.T) {
	db, qs := migrationDB(t)
	ctx := context.Background()
	before, err := LoadState(ctx, db)
	require.NoError(t, err)
	p, err := Preflight(ctx, db, qs)
	require.NoError(t, err)
	require.NoError(t, p.Validate())
	stager := &recordingStager{}
	r, err := Apply(ctx, db, qs, stager, p.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, "applied", r.Status)
	_, err = Verify(ctx, db)
	require.NoError(t, err)
	_, err = Apply(ctx, db, qs, stager, p.Fingerprint, true)
	require.NoError(t, err)
	require.Len(t, stager.versions, 1)
	after, err := LoadState(ctx, db)
	require.NoError(t, err)
	require.EqualValues(t, 12, after.PolicyVersion)
	r, err = Rollback(ctx, db, stager, p.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, "rolled_back", r.Status)
	restored, err := LoadState(ctx, db)
	require.NoError(t, err)
	require.EqualValues(t, 13, restored.PolicyVersion)
	restored.PolicyVersion = before.PolicyVersion
	require.Equal(t, before.Hash(), restored.Hash())
}
func TestApplyRollsBackWhenEventCannotBeStaged(t *testing.T) {
	db, qs := migrationDB(t)
	ctx := context.Background()
	before, err := LoadState(ctx, db)
	require.NoError(t, err)
	p, err := Preflight(ctx, db, qs)
	require.NoError(t, err)
	_, err = Apply(ctx, db, qs, &recordingStager{fail: true}, p.Fingerprint, true)
	require.ErrorContains(t, err, "outbox unavailable")
	after, err := LoadState(ctx, db)
	require.NoError(t, err)
	require.Equal(t, before.Hash(), after.Hash())
	var count int64
	require.NoError(t, db.Model(&Receipt{}).Count(&count).Error)
	require.Zero(t, count)
}
func TestApplyRefusesChangedClinicianAndRollbackRefusesLaterWrites(t *testing.T) {
	db, qs := migrationDB(t)
	ctx := context.Background()
	p, err := Preflight(ctx, db, qs)
	require.NoError(t, err)
	require.NoError(t, qs.Exec("UPDATE clinician SET is_active = 0 WHERE id = 100").Error)
	_, err = Apply(ctx, db, qs, &recordingStager{}, p.Fingerprint, true)
	require.ErrorContains(t, err, "facts changed")
	p, err = Preflight(ctx, db, qs)
	require.NoError(t, err)
	_, err = Apply(ctx, db, qs, &recordingStager{}, p.Fingerprint, true)
	require.NoError(t, err)
	require.NoError(t, db.Exec("UPDATE authz_roles SET description = 'later administrative write' WHERE name = ?", Operator).Error)
	_, err = Rollback(ctx, db, &recordingStager{}, p.Fingerprint, true)
	require.ErrorContains(t, err, "subsequent authorization writes")
}

func openMigrationTestDB(t *testing.T, suffix string) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("ROLE_MODEL_MYSQL_DSN")
	if dsn == "" {
		db, err := gorm.Open(sqlite.Open("file:"+t.Name()+suffix+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, err)
		return db
	}
	cfg, err := drivermysql.ParseDSN(dsn)
	require.NoError(t, err)
	cfg.DBName = ""
	cfg.ParseTime = true
	admin, err := gorm.Open(gormmysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	name := fmt.Sprintf("role_model_test_%d_%s", time.Now().UnixNano(), suffix)
	require.NoError(t, admin.Exec("CREATE DATABASE "+name+" CHARACTER SET utf8mb4").Error)
	cfg.DBName = name
	db, err := gorm.Open(gormmysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	t.Cleanup(func() {
		pool, _ := db.DB()
		if pool != nil {
			_ = pool.Close()
		}
		_ = admin.Exec("DROP DATABASE " + name).Error
		p, _ := admin.DB()
		if p != nil {
			_ = p.Close()
		}
	})
	return db
}

func TestApplyRefusesChangedNegativeClinicianEvidence(t *testing.T) {
	db, qs := migrationDB(t)
	ctx := context.Background()
	p, err := Preflight(ctx, db, qs)
	require.NoError(t, err)
	// User 2 still has no clinician, but the authoritative organization changed.
	require.NoError(t, qs.Exec("UPDATE staff SET org_id = 2 WHERE id = 20").Error)
	_, err = Apply(ctx, db, qs, &recordingStager{}, p.Fingerprint, true)
	require.ErrorContains(t, err, "preflight facts changed")
}

func TestPersistedPlanVerificationDetectsMissingGrant(t *testing.T) {
	db, qs := migrationDB(t)
	ctx := context.Background()
	p, err := Preflight(ctx, db, qs)
	require.NoError(t, err)
	_, err = Apply(ctx, db, qs, &recordingStager{}, p.Fingerprint, true)
	require.NoError(t, err)
	s, err := LoadState(ctx, db)
	require.NoError(t, err)
	for i, g := range s.Grants {
		if g.DeletedAt == nil && g.RevokedAt == nil {
			s.Grants = append(s.Grants[:i], s.Grants[i+1:]...)
			break
		}
	}
	require.ErrorContains(t, verifyPlan(s, p), "permission mismatch")
}

func TestApplyArchiveSurvivesTableRetirementMySQL(t *testing.T) {
	if os.Getenv("ROLE_MODEL_MYSQL_DSN") == "" {
		if os.Getenv("ROLE_MODEL_REQUIRE_MYSQL") == "true" {
			t.Fatal("ROLE_MODEL_MYSQL_DSN is required by this CI gate")
		}
		t.Skip("real MySQL required")
	}
	db, qs := migrationDB(t)
	ctx := context.Background()
	p, err := Preflight(ctx, db, qs)
	require.NoError(t, err)
	_, err = Apply(ctx, db, qs, &recordingStager{}, p.Fingerprint, true)
	require.NoError(t, err)
	archive, err := ArchiveInheritance(ctx, db, p.Fingerprint, true)
	require.NoError(t, err)
	require.Contains(t, archive.SchemaSQL, "CREATE TABLE")
	require.NoError(t, db.Exec("DROP TABLE authz_role_inheritances").Error)
	_, err = Verify(ctx, db)
	require.NoError(t, err)
	// Restoration starts with the original DDL and exact after-image; rollback
	_, err = Preflight(ctx, db, nil)
	require.NoError(t, err)
	_, err = Apply(ctx, db, nil, &recordingStager{fail: true}, p.Fingerprint, true)
	require.NoError(t, err)
	again, err := ArchiveInheritance(ctx, db, p.Fingerprint, true)
	require.NoError(t, err)
	require.Equal(t, archive.Checksum, again.Checksum)
	// then restores pre-migration facts and advances the policy version.
	require.NoError(t, db.Exec(archive.SchemaSQL).Error)
	var rows []LegacyEdge
	require.NoError(t, json.Unmarshal([]byte(archive.RowsJSON), &rows))
	require.NoError(t, db.Create(&rows).Error)
	_, err = Rollback(ctx, db, &recordingStager{}, p.Fingerprint, true)
	require.NoError(t, err)
}
