package rolemodel

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCompletedPreflightNeedsNoQSAndReplayDoesNotPublish(t *testing.T) {
	db, qs := migrationDB(t)
	ctx := context.Background()
	s, err := Status(ctx, db)
	require.NoError(t, err)
	require.Equal(t, "pending", s.State)
	p, err := Preflight(ctx, db, qs)
	require.NoError(t, err)
	stager := &recordingStager{}
	r, err := Apply(ctx, db, qs, stager, p.Fingerprint, true)
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		_, err = Preflight(ctx, db, nil)
		require.NoError(t, err)
		_, err = Apply(ctx, db, nil, stager, p.Fingerprint, true)
		require.NoError(t, err)
	}
	s, err = Status(ctx, db)
	require.NoError(t, err)
	require.Equal(t, "applied_unchanged", s.State)
	require.Equal(t, r.AfterHash, s.CurrentHash)
	require.Len(t, stager.versions, 1)
	require.NoError(t, db.Exec("UPDATE authz_roles SET description = 'later edit' WHERE name = ?", Operator).Error)
	s, err = Status(ctx, db)
	require.NoError(t, err)
	require.Equal(t, "applied_drifted", s.State)
	require.Contains(t, s.ChangedSections, "roles")
	_, err = Preflight(ctx, db, nil)
	require.Error(t, err)
	_, err = Apply(ctx, db, nil, stager, p.Fingerprint, true)
	require.Error(t, err)
	_, err = Verify(ctx, db)
	require.Error(t, err)
	require.Len(t, stager.versions, 1)
}

func TestStatusRejectsDamagedReceipt(t *testing.T) {
	db, qs := migrationDB(t)
	ctx := context.Background()
	p, err := Preflight(ctx, db, qs)
	require.NoError(t, err)
	_, err = Apply(ctx, db, qs, &recordingStager{}, p.Fingerprint, true)
	require.NoError(t, err)
	require.NoError(t, db.Model(&Receipt{}).Where("migration_id = ?", MigrationID).Update("after_json", "{}").Error)
	s, err := Status(ctx, db)
	require.Error(t, err)
	require.Equal(t, "invalid", s.State)
}
