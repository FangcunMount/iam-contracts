package handler

import (
	"context"
	"net/http/httptest"
	"testing"

	roleapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/testutil"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRoleDetailsHideProtectedRoles(t *testing.T) {
	fixture := testutil.NewFixture(t, nil)
	item, err := role.NewRole("reader", "Reader")
	require.NoError(t, err)
	require.NoError(t, fixture.Roles.Create(context.Background(), &item))
	protected, err := role.NewRole("platform_admin", "Platform", role.WithManagementProtection(role.ManagementProtected))
	require.NoError(t, err)
	require.NoError(t, fixture.Roles.Create(context.Background(), &protected))
	for _, tc := range []struct {
		id     string
		status int
	}{
		{item.ID.String(), 200}, {protected.ID.String(), 404}, {"987654321", 404},
	} {
		router := gin.New()
		router.GET("/:id", NewRoleHandler(nil, roleapp.NewRoleQueryService(fixture.Roles)).GetRole)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", "/"+tc.id, nil))
		require.Equal(t, tc.status, response.Code, response.Body.String())
	}
}
