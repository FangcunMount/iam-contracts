package messageprotocol

import (
	"context"
	"testing"

	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SQLite here checks the pure binding/selection policy only. Actual MySQL
// installation, original-root locking and missing-table errors are native tests.
func TestSelectionRejectsConflictingWriterAndRetainsHistoricalBinding(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	require.NoError(t, db.Exec(`CREATE TABLE iam_authz_message_protocol(owner TEXT PRIMARY KEY,protocol_type TEXT,store_id TEXT,writer_epoch TEXT,route_revision TEXT,generation INTEGER,schema_version INTEGER)`).Error)
	ctx := context.Background()
	b := transaction.Binding{Owner: "iam.mysql", StoreID: "primary", WriterEpoch: "m3-1"}
	require.ErrorIs(t, RequireOutbox(ctx, db, false), ErrSelection)
	require.ErrorIs(t, RequireTransaction(ctx, db, b, "m3-1", false), ErrSelection)
	require.NoError(t, db.Exec(`INSERT INTO iam_authz_message_protocol VALUES('iam.mysql','outbox','primary','m3-1','m3-1',1,1)`).Error)
	require.NoError(t, RequireOutbox(ctx, db, false), "retained SDK binding does not itself select transaction")
	require.ErrorIs(t, RequireTransaction(ctx, db, b, "m3-1", false), ErrSelection)
	require.NoError(t, db.Exec(`UPDATE iam_authz_message_protocol SET protocol_type='transaction',generation=2`).Error)
	require.NoError(t, RequireTransaction(ctx, db, b, "m3-1", false))
	require.ErrorIs(t, RequireOutbox(ctx, db, false), ErrSelection)
	for _, bad := range []transaction.Binding{
		{Owner: "IAM.mysql", StoreID: b.StoreID, WriterEpoch: b.WriterEpoch},
		{Owner: b.Owner, StoreID: "other", WriterEpoch: b.WriterEpoch},
		{Owner: b.Owner, StoreID: b.StoreID, WriterEpoch: "old"},
	} {
		require.ErrorIs(t, RequireTransaction(ctx, db, bad, "m3-1", false), ErrSelection)
	}
	require.ErrorIs(t, RequireTransaction(ctx, db, b, "M3-1", false), ErrSelection)
	for _, mutation := range []string{"schema_version=2", "generation=0", "protocol_type='unknown'"} {
		require.NoError(t, db.Exec("UPDATE iam_authz_message_protocol SET "+mutation).Error)
		require.ErrorIs(t, RequireTransaction(ctx, db, b, "m3-1", false), ErrSelection)
		require.ErrorIs(t, RequireOutbox(ctx, db, false), ErrSelection)
		require.NoError(t, db.Exec("UPDATE iam_authz_message_protocol SET schema_version=1,generation=2,protocol_type='transaction'").Error)
	}
	require.ErrorIs(t, RequireOutbox(nil, db, false), ErrSelection)
	require.ErrorIs(t, RequireTransaction(ctx, nil, b, "m3-1", false), ErrSelection)
}
