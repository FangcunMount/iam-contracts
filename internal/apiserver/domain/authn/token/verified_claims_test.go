package token

import (
	"testing"
	"time"

	"github.com/FangcunMount/iam/v4/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestNewAccessTokenClaimsEnforcesUserSessionIdentity(t *testing.T) {
	now := time.Now().UTC()
	claims, err := NewAccessTokenClaims(AccessTokenClaims{
		TokenID: "jti", Subject: "1", SessionID: "sid", UserID: meta.FromUint64(1), LoginIdentityID: meta.FromUint64(2),
		Issuer: "iam", Audience: []string{"qs-api"}, IssuedAt: now, NotBefore: now, ExpiresAt: now.Add(time.Minute),
	})
	require.NoError(t, err)
	require.Equal(t, TokenTypeAccess, claims.TokenType)

	_, err = NewAccessTokenClaims(AccessTokenClaims{
		TokenID: "jti", Subject: "different", SessionID: "sid", UserID: meta.FromUint64(1), LoginIdentityID: meta.FromUint64(2),
		Issuer: "iam", Audience: []string{"qs-api"}, IssuedAt: now, NotBefore: now, ExpiresAt: now.Add(time.Minute),
	})
	require.ErrorContains(t, err, "sub must equal user_id")
}

func TestAccessClaimsRejectExplicitNonAccessType(t *testing.T) {
	now := time.Now().UTC()
	for _, kind := range []TokenType{TokenTypeRefresh, TokenType("unknown")} {
		claims, err := NewAccessTokenClaims(AccessTokenClaims{
			TokenID: "jti", TokenType: kind, Subject: "1", UserID: meta.FromUint64(1), LoginIdentityID: meta.FromUint64(2), SessionID: "sid",
			Issuer: "iam", Audience: []string{"iam-api"}, IssuedAt: now, NotBefore: now, ExpiresAt: now.Add(time.Minute),
		})
		require.Error(t, err)
		require.Nil(t, claims)
	}
}
