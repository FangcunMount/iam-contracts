package migration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestAuthorizationRetirementUpgradeMySQL(t *testing.T) {
	ctx := context.Background()
	pool := openMigrationMySQL(t)
	database := migrationEnvOr("MYSQL_DATABASE", "iam_test")
	var count int
	require.NoError(t, pool.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=?", database).Scan(&count))
	require.Zero(t, count, "requires dedicated empty database")
	_, _, err := NewMigrator(pool, &Config{Enabled: true, Database: database}).RunTo(31)
	require.NoError(t, err)
	db, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: openMigrationMySQL(t)}), &gorm.Config{})
	require.NoError(t, err)
	// 历史平台列表授权必须保留普通能力并补充全量能力，角色 ID 不变。
	require.NoError(t, db.Exec(`INSERT INTO authz_permission_grants(id,tenant_id,role_id,resource_id,resource_pattern,action,constraint_set,grant_key,granted_by,granted_at)
 SELECT 990001,'platform',900000001,id,`+"`key`"+`,'list','{"version":1,"all_of":[]}',
 SHA2(CONCAT('v1',CHAR(0),'platform',CHAR(0),'900000001',CHAR(0),id,CHAR(0),`+"`key`"+`,CHAR(0),'list',CHAR(0),'{"version":1,"all_of":[]}'),256),'test',NOW()
 FROM authz_resources WHERE `+"`key`"+`='iam:identity:collection:profiles' AND deleted_at IS NULL`).Error)
	before, err := AnalyzeTenantRetirement(ctx, db)
	require.NoError(t, err)
	require.True(t, before.Ready, before.Issues)
	require.NoError(t, db.Exec("UPDATE authz_roles SET tenant_id='unknown' WHERE id=2").Error)
	invalid, err := AnalyzeTenantRetirement(ctx, db)
	require.NoError(t, err)
	require.False(t, invalid.Ready)
	_, err = PrepareTenantRetirement(ctx, db, invalid.Fingerprint)
	require.Error(t, err)
	var name string
	require.NoError(t, db.Raw("SELECT name FROM authz_roles WHERE id=2").Scan(&name).Error)
	require.Equal(t, "tenant_admin", name)
	require.NoError(t, db.Exec("UPDATE authz_roles SET tenant_id='fangcun' WHERE id=2").Error)
	before, err = AnalyzeTenantRetirement(ctx, db)
	require.NoError(t, err)
	after, err := PrepareTenantRetirement(ctx, db, before.Fingerprint)
	require.NoError(t, err)
	require.True(t, after.Ready)
	repeated, err := PrepareTenantRetirement(ctx, db, before.Fingerprint)
	require.NoError(t, err)
	require.Equal(t, after.Fingerprint, repeated.Fingerprint)
	// 准备之后的新写入必须使迁移凭据失效。
	require.NoError(t, db.Exec("UPDATE authz_roles SET description='changed' WHERE id=2").Error)
	var markers int64
	require.NoError(t, db.Table("iam_authorization_retirement_preparation").Count(&markers).Error)
	require.Zero(t, markers)
	before, err = AnalyzeTenantRetirement(ctx, db)
	require.NoError(t, err)
	_, err = PrepareTenantRetirement(ctx, db, before.Fingerprint)
	require.NoError(t, err)
	version, changed, err := NewMigrator(openMigrationMySQL(t), &Config{Enabled: true, Database: database}).RunTo(32)
	require.NoError(t, err)
	require.True(t, changed)
	require.EqualValues(t, 32, version)
	version, changed, err = NewMigrator(openMigrationMySQL(t), &Config{Enabled: true, Database: database}).RunTo(32)
	require.NoError(t, err)
	require.False(t, changed)
	require.EqualValues(t, 32, version)
	var actions []string
	require.NoError(t, db.Table("authz_permission_grants").Where("role_id=900000001").Order("action").Pluck("action", &actions).Error)
	require.Equal(t, []string{"list", "list_all"}, actions)
	require.NoError(t, db.Raw("SELECT name FROM authz_roles WHERE id=900000001").Scan(&name).Error)
	require.Equal(t, "platform_admin", name)
	require.NoError(t, db.Raw("SELECT name FROM authz_roles WHERE id=2").Scan(&name).Error)
	require.Equal(t, "iam_admin", name)
	var columns int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND COLUMN_NAME='tenant_id'").Scan(&columns).Error)
	require.Zero(t, columns)
	var current int64
	require.NoError(t, db.Raw("SELECT policy_version FROM authz_policy_versions WHERE id=1").Scan(&current).Error)
	require.Equal(t, before.InitialPolicyVersion, current)
}
