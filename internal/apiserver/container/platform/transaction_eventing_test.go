package platform

import (
	"context"
	"database/sql"
	"errors"
	hostuow "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/uow/authz"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/FangcunMount/reliable-messaging/transport"
	"github.com/stretchr/testify/require"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"testing"
	"time"
)

type normalPeerStub struct {
	startErr                  error
	starts, closes, publishes int
}

func (s *normalPeerStub) Start() error                { s.starts++; return s.startErr }
func (s *normalPeerStub) Close(context.Context) error { s.closes++; return nil }
func (s *normalPeerStub) PublishRaw(context.Context, string, []byte) transport.Result {
	s.publishes++
	return transport.Result{Outcome: transport.Confirmed}
}
func ownerFixture(t *testing.T, driver transactionDriver, normal normalTransactionPeer) *TransactionEventing {
	t.Helper()
	runtime := &TransactionRuntime{sender: driver, admission: true}
	db, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: new(sql.DB), SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	local, err := hostuow.NewTransactionUnitOfWork(db, nil, transaction.Binding{Owner: "iam.mysql", StoreID: "fixture", WriterEpoch: "m3-1"})
	require.NoError(t, err)
	owner := &TransactionEventing{runtime: runtime, normal: normal, startDone: make(chan struct{}), cleanupBudget: time.Second}
	topic := "iam.authz.version.v2"
	physical, err := transport.PhysicalTopicV2(topic, transport.TransactionMessageType, "iam.mysql", "m3-1")
	require.NoError(t, err)
	owner.Writer, err = hostuow.NewPlannedUnitOfWork(local, runtime, hostuow.PlanConfig{LogicalTopic: topic, PhysicalTopic: physical, Sends: 8, ProofBudget: time.Second, Admission: owner.admitted})
	require.NoError(t, err)
	return owner
}
func TestTransactionEventOwnerPausesBothPublishTypesAndKeepsChecker(t *testing.T) {
	ctx := context.Background()
	driver := &transactionDriverStub{}
	normal := &normalPeerStub{}
	owner := ownerFixture(t, driver, normal)
	require.ErrorIs(t, owner.SetAdmission(true), ErrTransactionAdmissionClosed)
	require.Equal(t, transport.Rejected, owner.PublishRaw(ctx, "sms", nil).Outcome)
	require.NoError(t, owner.Start(ctx))
	require.Equal(t, 1, normal.starts)
	require.Equal(t, transport.Confirmed, owner.PublishRaw(ctx, "sms", nil).Outcome)
	require.NoError(t, owner.SetAdmission(false))
	require.Equal(t, transport.Rejected, owner.PublishRaw(ctx, "sms", nil).Outcome)
	require.Zero(t, driver.stopCalls)
	require.NoError(t, owner.SetAdmission(true))
	require.NoError(t, owner.Stop(ctx))
	require.Equal(t, 1, normal.closes)
	require.ErrorIs(t, owner.SetAdmission(true), ErrTransactionAdmissionClosed)
}
func TestTransactionEventOwnerCleansStartupFailureWithoutClosingHostPool(t *testing.T) {
	driver := &transactionDriverStub{}
	normal := &normalPeerStub{startErr: errors.New("startup rejected")}
	owner := ownerFixture(t, driver, normal)
	require.ErrorContains(t, owner.Start(context.Background()), "startup rejected")
	require.Equal(t, 1, driver.closeCalls)
	require.Equal(t, 1, normal.closes)
	require.False(t, owner.admitted())
	require.NoError(t, owner.Writer.Wait(context.Background()))
}

type blockedTransactionDriver struct {
	transactionDriverStub
	entered, release chan struct{}
}

func (s *blockedTransactionDriver) Start(ctx context.Context) error {
	close(s.entered)
	<-s.release
	return s.transactionDriverStub.Start(ctx)
}
func TestTransactionEventOwnerReportsActualStartupOwnershipAtShutdownDeadline(t *testing.T) {
	driver := &blockedTransactionDriver{entered: make(chan struct{}), release: make(chan struct{})}
	normal := &normalPeerStub{}
	owner := ownerFixture(t, driver, normal)
	done := make(chan error, 1)
	go func() { done <- owner.Start(context.Background()) }()
	<-driver.entered
	expired, cancel := context.WithCancel(context.Background())
	cancel()
	var incomplete *transaction.ShutdownIncomplete
	require.ErrorAs(t, owner.Stop(expired), &incomplete)
	require.True(t, incomplete.RuntimePending)
	close(driver.release)
	require.ErrorIs(t, <-done, ErrTransactionAdmissionClosed)
	require.Zero(t, normal.starts, "startup launched another client after stopping")
	require.NoError(t, owner.Stop(context.Background()))
}
