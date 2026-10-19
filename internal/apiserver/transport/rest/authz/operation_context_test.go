package authz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOperationContextPreservesLegacyAndRejectsAmbiguousKeys(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		keys    []string
		status  int
	}{{"legacy", false, nil, 200}, {"unsupported-key", false, []string{"key"}, 503}, {"optional", true, nil, 200}, {"keyed", true, []string{"original"}, 200}, {"empty", true, []string{""}, 400}, {"two", true, []string{"a", "b"}, 400}, {"joined", true, []string{"a,b"}, 400}, {"space", true, []string{" a"}, 400}} {
		t.Run(tc.name, func(t *testing.T) {
			engine := gin.New()
			called := false
			id := "f46ebdfd-c2a4-43b9-aafb-b98b9675505d"
			engine.POST("/write", operationContext(tc.enabled), func(c *gin.Context) {
				called = true
				key, exists := change.RequestKey(c.Request.Context())
				if len(tc.keys) != 0 {
					require.True(t, exists)
					require.Equal(t, tc.keys[0], key)
				} else {
					require.False(t, exists)
				}
				operation.ReportReference(c.Request.Context(), "untrusted-invalid-id")
				require.Empty(t, c.Writer.Header().Get("Operation-ID"))
				operation.ReportReference(c.Request.Context(), id)
				c.JSON(200, gin.H{"legacy": true})
			})
			request := httptest.NewRequest(http.MethodPost, "/write", nil)
			for _, key := range tc.keys {
				request.Header.Add("Idempotency-Key", key)
			}
			request.Header.Set("Operation-ID", "client-cannot-select-id")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			require.Equal(t, tc.status, response.Code)
			require.Equal(t, tc.status == 200, called)
			if tc.enabled && called {
				require.Equal(t, id, response.Header().Get("Operation-ID"))
				require.Equal(t, "/api/v4/authz/operations/"+id, response.Header().Get("Operation-Location"))
			} else {
				require.Empty(t, response.Header().Get("Operation-ID"))
			}
		})
	}
}
