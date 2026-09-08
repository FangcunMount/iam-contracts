package token

import (
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestTokenKindsAreExpressedByDistinctDomainTypes(t *testing.T) {
	t.Parallel()

	access := NewAccessToken("a", "access", "sid", meta.FromUint64(1), meta.FromUint64(2), time.Now(), time.Now().Add(time.Minute))
	refresh := NewRefreshToken("r", "refresh", "sid", meta.FromUint64(1), meta.FromUint64(2), time.Now(), time.Now().Add(time.Hour))

	require.Equal(t, TokenTypeAccess, access.Kind())
	require.Equal(t, TokenTypeRefresh, refresh.Kind())
	require.Equal(t, "sid", access.SessionID)
	require.Equal(t, "sid", refresh.SessionID)
}

func TestTokenMetadataUsesExplicitTimeForExpiryDecisions(t *testing.T) {
	t.Parallel()

	expiresAt := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	metadata := TokenMetadata{ExpiresAt: expiresAt}

	require.False(t, metadata.IsExpiredAt(expiresAt))
	require.True(t, metadata.IsExpiredAt(expiresAt.Add(time.Nanosecond)))
	require.Equal(t, time.Minute, metadata.RemainingAt(expiresAt.Add(-time.Minute)))
}
