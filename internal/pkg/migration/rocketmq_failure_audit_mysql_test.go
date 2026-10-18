package migration

import (
	"context"
	"os"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/messagefailure"
	"github.com/FangcunMount/reliable-messaging/transport"
	"github.com/stretchr/testify/require"
)

func TestIAMRocketMQAuditMigrationPreservesFailuresMySQL(t *testing.T) {
	if os.Getenv("IAM_RM_MIGRATION_REQUIRED") == "1" {
		require.NotEmpty(t, os.Getenv("MYSQL_HOST"))
	}
	db := openMigrationMySQL(t)
	var tables int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE()").Scan(&tables))
	require.Zero(t, tables, "dedicated empty database required")
	up, err := migrations.ReadFile("migrations/000041_message_failure_audit.up.sql")
	require.NoError(t, err)
	down, err := migrations.ReadFile("migrations/000041_message_failure_audit.down.sql")
	require.NoError(t, err)
	_, err = db.Exec(string(up))
	require.NoError(t, err)
	_, err = db.Exec(string(down))
	require.NoError(t, err, "empty audit can be rolled back")
	_, err = db.Exec(string(up))
	require.NoError(t, err)
	audit, err := messagefailure.New(db)
	require.NoError(t, err)
	require.NoError(t, audit.RecordRocketMQ(context.Background(), transport.Failure{
		Provider: "rocketmq", Cause: "terminal handler failure",
		Message: transport.Received{Provider: "rocketmq", Topic: "iam.authz.version.v2",
			PhysicalTopic: transport.PhysicalTopic("iam.authz.version.v2"), Channel: "iam-policy-sync.test",
			ID: "original-id", TransportID: "broker-id", Attempts: 8,
			Wire: []byte("immutable-envelope"), Payload: []byte("original-payload")},
	}))
	_, err = db.Exec(string(down))
	require.Error(t, err, "application rollback must retain terminal failures")
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM iam_message_failure_audit").Scan(&count))
	require.Equal(t, 1, count)
}
