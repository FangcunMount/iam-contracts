package authz

import (
	"context"
	"database/sql"
	"testing"
	"time"

	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	"github.com/FangcunMount/iam/v5/internal/apiserver/testhelpers"
	database "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/pkg/event"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestOriginalTransactionConstructorIsInertAndRejectsUnboundHandles(t *testing.T) {
	binding := transaction.Binding{Owner: "iam.mysql", StoreID: "owned-primary", WriterEpoch: "m3-1"}
	// A zero SQL DB cannot run queries. Constructors must neither ping nor do DDL.
	pool := new(sql.DB)
	db, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: pool, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	local, err := NewTransactionUnitOfWork(db, nil, binding)
	require.NoError(t, err)
	require.Equal(t, binding, local.Store().Binding())
	_, err = NewTransactionUnitOfWork(testhelpers.SetupTempSQLiteDB(t), nil, binding)
	require.ErrorIs(t, err, transaction.ErrTransactionRequired)
	_, err = NewTransactionUnitOfWork(nil, nil, binding)
	require.ErrorIs(t, err, transaction.ErrTransactionRequired)
}
func TestTransactionStagerRequiresExactlyOneFrozenEventAndLiveRoot(t *testing.T) {
	intent, err := change.NewCommandIntent("role.update", "user:7", "global", "key", map[string]string{"id": "42"})
	require.NoError(t, err)
	at := time.Date(2026, 10, 19, 14, 0, 0, 0, time.UTC)
	frozen, err := change.FreezeVersionPlan(intent, 41, "operation", "event", at, "iam.authz.version.v2", "physical")
	require.NoError(t, err)
	plan, err := change.DecodeVersionPlan(frozen)
	require.NoError(t, err)
	proof := &eventProof{plan: plan, active: true}
	exact := event.Event[domain.VersionChangedPayload]{BaseEvent: event.BaseEvent{ID: "event", EventTypeValue: plan.Message.Input().EventType, OccurredAtValue: at, AggregateTypeValue: "PolicyVersion", AggregateIDValue: "42"}, Data: domain.VersionChangedPayload{Version: 42}}
	stager := transactionEventStager{}
	require.ErrorIs(t, stager.Stage(context.Background(), exact), transaction.ErrTransactionRequired)
	db := testhelpers.SetupTempSQLiteDB(t)
	require.NoError(t, database.NewUnitOfWork(db).WithinTransaction(context.Background(), func(ctx context.Context) error {
		ctx = context.WithValue(ctx, eventProofKey{}, proof)
		wrong := exact
		wrong.Data.Version = 43
		require.ErrorIs(t, stager.Stage(ctx, wrong), change.ErrInvalidChangePlan)
		require.False(t, proof.exactlyOnce())
		require.NoError(t, stager.Stage(ctx, exact))
		require.True(t, proof.exactlyOnce())
		require.ErrorIs(t, stager.Stage(ctx, exact), change.ErrInvalidChangePlan)
		proof.close()
		require.False(t, proof.exactlyOnce())
		require.ErrorIs(t, stager.Stage(ctx, exact), change.ErrInvalidChangePlan)
		return nil
	}))
}
