package authz

import (
	"net/http"
	"strings"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	"github.com/gin-gonic/gin"
)

// operationContext runs after authentication. Optional keys apply only to the
// reviewed authorization writers, and never turn a read into a reservation.
func operationContext(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut && c.Request.Method != http.MethodDelete {
			c.Next()
			return
		}
		ctx := c.Request.Context()
		keys := c.Request.Header.Values("Idempotency-Key")
		if len(keys) != 0 {
			if !enabled {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"code": "operation_journal_unavailable"})
				return
			}
			if len(keys) != 1 || strings.ContainsAny(keys[0], ",\r\n\x00") {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": "invalid_idempotency_key"})
				return
			}
			var err error
			ctx, err = change.WithRequestKey(ctx, keys[0])
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": "invalid_idempotency_key"})
				return
			}
		}
		if enabled {
			// The application reports a validated reference before the handler writes
			// its original success body. No request header can choose this identity.
			ctx, _ = operation.WithReferenceSink(ctx, func(id string) {
				c.Header("Operation-ID", id)
				c.Header("Operation-Location", "/api/v4/authz/operations/"+id)
			})
		}
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
