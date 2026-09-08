// Package authz enforces authorization decisions at the HTTP boundary.
package authz

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/requestctx"
	"github.com/FangcunMount/iam/v5/pkg/core"
	"github.com/gin-gonic/gin"
)

// RoutePermissionChecker answers a route-level permission question without
// controlling the HTTP request lifecycle.
type RoutePermissionChecker interface {
	CheckRoutePermission(ctx context.Context, subjectKey, resourceKey, action string) (bool, error)
}

// Middleware turns authorization decisions into HTTP allow/deny execution.
type Middleware struct {
	checker RoutePermissionChecker
}

func NewMiddleware(checker RoutePermissionChecker) *Middleware {
	return &Middleware{checker: checker}
}

func (m *Middleware) Available() bool {
	return m != nil && m.checker != nil
}

// RequirePermission 在单一授权空间执行资源和动作检查。
func (m *Middleware) RequirePermission(resourceKey, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.Available() {
			recordHTTPAuthorization(resourceKey, action, "error")
			core.WriteResponse(c, errors.WithCode(code.ErrInternalServerError, "Authorization engine not configured"), nil)
			c.Abort()
			return
		}
		userID, ok := requestctx.UserID(c)
		if !ok {
			recordHTTPAuthorization(resourceKey, action, "unauthenticated")
			core.WriteResponse(c, errors.WithCode(code.ErrTokenInvalid, "Not authenticated"), nil)
			c.Abort()
			return
		}
		allowed, err := m.checker.CheckRoutePermission(c.Request.Context(), "user:"+userID.String(), resourceKey, action)
		if err != nil {
			recordHTTPAuthorization(resourceKey, action, "error")
			core.WriteResponse(c, authorizationError(err), nil)
			c.Abort()
			return
		}
		if !allowed {
			recordHTTPAuthorization(resourceKey, action, "denied")
			core.WriteResponse(c, errors.WithCode(code.ErrPermissionDenied, "Forbidden"), nil)
			c.Abort()
			return
		}
		recordHTTPAuthorization(resourceKey, action, "allowed")
		c.Next()
	}
}

func authorizationError(err error) error {
	if errors.IsCode(err, code.ErrAuthorizationPolicyUnavailable) {
		return err
	}
	return errors.WithCode(code.ErrInternalServerError, "Authorization check failed")
}
