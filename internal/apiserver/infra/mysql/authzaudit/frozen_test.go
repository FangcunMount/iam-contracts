package authzaudit

import (
	"context"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	base "github.com/FangcunMount/iam/v5/internal/pkg/database/mysql"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestFrozenAuditReadsExactPlanAndNeverFallsBackToNewIDs(t *testing.T) {
	command, err := change.NewCommandIntent("role.create", "user:7", "global", "key", map[string]string{"name": "frozen"})
	require.NoError(t, err)
	at := time.Date(2026, 10, 19, 14, 0, 0, 0, time.UTC)
	key := change.ArtifactKey("role", "frozen")
	plan, err := change.FreezeVersionPlanWithArtifacts(command, 41, "operation", "event", at, "logical", "physical", change.LocalArtifacts{AuditAt: at, IDs: map[string]meta.ID{key: 77}})
	require.NoError(t, err)
	e := transaction.Execution{Plan: plan, Fence: 1, WriterEpoch: "m3-1", LeaseUntil: at.Add(time.Minute)}
	e.Token[0] = 1
	ctx, err := change.WithExecutionPlan(context.Background(), e, command)
	require.NoError(t, err)
	row := base.AuditFields{}
	handled, err := Create(ctx, key, &row)
	require.NoError(t, err)
	require.True(t, handled)
	require.Equal(t, meta.ID(77), row.ID)
	require.Equal(t, at, row.CreatedAt)
	require.Equal(t, at, row.UpdatedAt)
	untouched := base.AuditFields{}
	handled, err = Create(ctx, change.ArtifactKey("role", "other"), &untouched)
	require.ErrorIs(t, err, change.ErrInvalidChangePlan)
	require.True(t, handled)
	require.Zero(t, untouched.ID)
	clock, err := Time(ctx)
	require.NoError(t, err)
	require.Equal(t, at, clock)
	id, err := ID(ctx, key)
	require.NoError(t, err)
	require.Equal(t, row.ID, id)
	handled, err = Update(ctx, &row)
	require.NoError(t, err)
	require.True(t, handled)
	require.Equal(t, at, row.UpdatedAt)
}
