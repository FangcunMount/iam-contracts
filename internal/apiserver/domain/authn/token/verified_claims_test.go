package token

import (
	"testing"
	"time"

	"github.com/FangcunMount/iam/v4/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestNewVerifiedUserTokenClaimsEnforcesUserSessionIdentity(t *testing.T) {
	now := time.Now().UTC()
	claims, err := NewVerifiedUserTokenClaims(VerifiedTokenClaims{
		TokenID: "jti", Subject: "1", SessionID: "sid", UserID: meta.FromUint64(1), LoginIdentityID: meta.FromUint64(2),
		Issuer: "iam", Audience: []string{"qs-api"}, IssuedAt: now, NotBefore: now, ExpiresAt: now.Add(time.Minute),
	})
	require.NoError(t, err)
	require.Equal(t, TokenTypeAccess, claims.TokenType)

	_, err = NewVerifiedUserTokenClaims(VerifiedTokenClaims{
		TokenID: "jti", Subject: "different", SessionID: "sid", UserID: meta.FromUint64(1), LoginIdentityID: meta.FromUint64(2),
		Issuer: "iam", Audience: []string{"qs-api"}, IssuedAt: now, NotBefore: now, ExpiresAt: now.Add(time.Minute),
	})
	require.ErrorContains(t, err, "sub must equal user_id")
}
