package requestctx

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	tokenapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authn/token"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

func TestBusinessOrgIDFromOrgClaimOnly(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	t.Run("org_id claim", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		SetClaims(c, &tokenapp.TokenClaims{OrgID: meta.FromUint64(42)})
		id, ok := BusinessOrgID(c)
		require.True(t, ok)
		require.Equal(t, uint64(42), id)
	})

	t.Run("numeric tenant domain does not imply org", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		SetClaims(c, &tokenapp.TokenClaims{})
		_, ok := BusinessOrgID(c)
		require.False(t, ok)
	})
}
