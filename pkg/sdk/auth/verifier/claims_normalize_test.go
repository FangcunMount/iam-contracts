package verifier

import (
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/require"
)

func TestApplyOrg(t *testing.T) {
	t.Parallel()

	t.Run("new token domain and org", func(t *testing.T) {
		claims := &TokenClaims{}
		applyOrg(claims, "42")
		require.Equal(t, "42", claims.OrgID)
	})

	t.Run("legacy numeric tenant does not infer org", func(t *testing.T) {
		claims := &TokenClaims{}
		applyOrg(claims, "")
		require.Empty(t, claims.OrgID)
	})
}

func TestExtractClaimsOrgID(t *testing.T) {
	t.Parallel()

	token := jwt.New()
	require.NoError(t, token.Set("tenant_id", "fangcun"))
	require.NoError(t, token.Set("org_id", "42"))

	claims := extractClaims(token)
	require.Equal(t, "42", claims.OrgID)
	orgID, ok := claims.BusinessOrgID()
	require.True(t, ok)
	require.Equal(t, uint64(42), orgID)
}

func TestExtractClaimsLegacyNumericTenantDoesNotInferOrg(t *testing.T) {
	t.Parallel()

	token := jwt.New()
	require.NoError(t, token.Set("tenant_id", "1"))

	claims := extractClaims(token)
	require.Empty(t, claims.OrgID)
	_, ok := claims.BusinessOrgID()
	require.False(t, ok)
}

func TestApplyOrgFromRemoteFields(t *testing.T) {
	t.Parallel()

	claims := &TokenClaims{UserID: "1001"}
	applyOrg(claims, "42")
	orgID, ok := claims.BusinessOrgID()
	require.True(t, ok)
	require.Equal(t, uint64(42), orgID)
}
