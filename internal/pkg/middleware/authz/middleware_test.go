package authz

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/FangcunMount/iam/v5/internal/pkg/requestctx"
	"github.com/gin-gonic/gin"
)

type routePermissionCheckerStub struct {
	allowed bool
	err     error
}

func (s routePermissionCheckerStub) CheckRoutePermission(_ context.Context, _, _, _ string) (bool, error) {
	return s.allowed, s.err
}

func TestRequirePermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		checker    routePermissionCheckerStub
		withUser   bool
		wantStatus int
	}{

		{name: "allowed", checker: routePermissionCheckerStub{allowed: true}, withUser: true, wantStatus: http.StatusNoContent},
		{name: "denied", withUser: true, wantStatus: http.StatusForbidden},
		{name: "checker failure", checker: routePermissionCheckerStub{err: errors.New("unavailable")}, withUser: true, wantStatus: http.StatusInternalServerError},
		{name: "expired policy", checker: routePermissionCheckerStub{err: perrors.WithCode(code.ErrAuthorizationPolicyUnavailable, "expired")}, withUser: true, wantStatus: http.StatusServiceUnavailable},
		{name: "missing principal", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gin.SetMode(gin.TestMode)
			engine := gin.New()
			if tc.withUser {
				engine.Use(func(c *gin.Context) {
					requestctx.SetUserID(c, meta.FromUint64(10001))
					c.Next()
				})
			}
			middleware := NewMiddleware(tc.checker)
			engine.GET("/protected",
				middleware.RequirePermission("iam:authz:collection:roles", "read"),
				func(c *gin.Context) { c.Status(http.StatusNoContent) },
			)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, req)
			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tc.wantStatus, recorder.Body.String())
			}
		})
	}
}
