package token

import (
	"context"
	perrors "github.com/FangcunMount/component-base/pkg/errors"
	sessiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
	"github.com/FangcunMount/iam/v4/internal/pkg/code"
	"github.com/FangcunMount/iam/v4/internal/pkg/meta"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type capturingEncoder struct{ claims *AccessTokenClaims }

func (e *capturingEncoder) EncodeAccessToken(_ context.Context, c *AccessTokenClaims) (string, error) {
	e.claims = c
	return "signed", nil
}

type fixedRefreshExpiry struct {
	now     time.Time
	expires time.Time
}

func (e *fixedRefreshExpiry) NextRefreshExpiresAt(now time.Time, _ *sessiondomain.Session) (time.Time, error) {
	e.now = now
	return e.expires, nil
}

func TestMintUsesOneClockReadingAndPreservesRefreshBoundary(t *testing.T) {
	now := time.Date(2026, 9, 8, 10, 0, 0, 765432100, time.UTC)
	calls := 0
	encoder := &capturingEncoder{}
	expiry := &fixedRefreshExpiry{expires: now.Add(7 * time.Hour)}
	minter := newTokenSetMinter(encoder, expiry, IssuanceConfig{Issuer: "iam", Audience: []string{"iam-api"}, AccessTTL: 15 * time.Minute}, func() time.Time { calls++; return now })
	sess := &sessiondomain.Session{SessionID: "sid", UserID: meta.FromUint64(1), LoginIdentityID: meta.FromUint64(2)}
	result, err := minter.MintTokenSet(context.Background(), sess)
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Equal(t, now.Truncate(time.Second), result.AccessToken.IssuedAt)
	require.Equal(t, encoder.claims.TokenID, result.AccessToken.ID)
	require.Equal(t, encoder.claims.IssuedAt, result.AccessToken.IssuedAt)
	require.Equal(t, encoder.claims.ExpiresAt, result.AccessToken.ExpiresAt)
	require.Equal(t, now, expiry.now)
	require.Equal(t, expiry.expires, result.RefreshToken.ExpiresAt)
	require.Empty(t, result.RefreshToken.AMR)
	require.Empty(t, result.RefreshToken.SessionClaims)
}

func TestAudienceValidationPrecedesCryptographyAndState(t *testing.T) {
	// Nil dependencies make any accidental downstream call fail the test.
	for _, audience := range [][]string{nil, {}, {""}, {"qs-api", " "}} {
		_, err := newVerifier(nil, nil, nil, nil).VerifyToken(context.Background(), "token", audience)
		require.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
	}
	normalized, err := NormalizeExpectedAudience([]string{" qs-api ", "qs-api", "collection-api"})
	require.NoError(t, err)
	require.Equal(t, []string{"qs-api", "collection-api"}, normalized)
	codec := &signatureVerifierStub{claims: &AccessTokenClaims{TokenType: TokenTypeAccess, Audience: []string{"qs-api"}}}
	_, err = newVerifier(codec, nil, nil, nil).VerifyToken(context.Background(), "token", []string{"iam-api"})
	require.True(t, perrors.IsCode(err, code.ErrTokenInvalid))
	require.True(t, matchesAudience([]string{"qs-api"}, []string{"collection-api", "qs-api"}))
}
