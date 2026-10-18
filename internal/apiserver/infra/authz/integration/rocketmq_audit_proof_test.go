//go:build reliable_messaging

package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/messagefailure"
	"github.com/FangcunMount/iam/v5/internal/apiserver/testfixtures/authzdb"
	"github.com/FangcunMount/reliable-messaging/transport"
	"github.com/stretchr/testify/require"
)

// Required storage proof, not a Broker or business-consumption acceptance test.
func TestReliableMessagingRocketMQFailureAudit(t *testing.T) {
	if os.Getenv("IAM_AUTHZ_TEST_MYSQL_DSN") == "" || os.Getenv("RM_IAM_ROCKETMQ_AUDIT_SCHEMA") == "" {
		t.Fatal("disposable MySQL and tracked RocketMQ audit migration required; must not skip")
	}
	db := authzdb.Open(t, true)
	schema, err := os.ReadFile(os.Getenv("RM_IAM_ROCKETMQ_AUDIT_SCHEMA"))
	require.NoError(t, err)
	require.NoError(t, db.Exec(string(schema)).Error)
	pool, err := db.DB()
	require.NoError(t, err)
	audit, err := messagefailure.New(pool)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	require.NoError(t, messagefailure.CheckRocketMQAudit(ctx, pool))
	failure := transport.Failure{Provider: "rocketmq", Cause: "handler failed at delivery limit", Message: transport.Received{
		Provider: "rocketmq", ID: "original-policy-event", TransportID: "physical-one",
		Topic: "iam.authz.version.v2", Channel: "iam-policy-sync.test-instance",
		PhysicalTopic: transport.PhysicalTopic("iam.authz.version.v2"),
		Metadata:      map[string]string{"event_type": "iam.authz.version_changed.v2"},
		Payload:       []byte(`{"version":7}`), Wire: []byte("immutable-original-envelope"), Attempts: 8,
	}}
	require.NoError(t, audit.RecordRocketMQ(ctx, failure))
	failure.Message.TransportID = "physical-two"
	require.NoError(t, audit.RecordRocketMQ(ctx, failure))
	var count, seen int
	require.NoError(t, pool.QueryRowContext(ctx, "SELECT COUNT(*),MAX(seen_count) FROM iam_message_failure_audit").Scan(&count, &seen))
	require.Equal(t, 1, count)
	require.Equal(t, 2, seen)
	failure.Message.Wire = []byte("changed-original-envelope")
	require.Error(t, audit.RecordRocketMQ(ctx, failure), "identity collision must not authorize ACK")
	failure.Message.ID = ""
	failure.Message.TransportID = "malformed-physical"
	failure.Message.Wire = []byte("malformed original bytes")
	require.NoError(t, audit.RecordRocketMQ(ctx, failure))
	var applicationID, provider string
	require.NoError(t, pool.QueryRowContext(ctx,
		"SELECT application_id,provider FROM iam_message_failure_audit WHERE last_transport_id='malformed-physical'").Scan(&applicationID, &provider))
	require.Empty(t, applicationID, "physical identity must not be invented as application identity")
	require.Equal(t, "rocketmq", provider)
	require.NoError(t, db.Exec("DROP TABLE iam_message_failure_audit").Error)
	require.Error(t, audit.RecordRocketMQ(ctx, failure), "unavailable audit store cannot authorize ACK")
}
