package authz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	authzapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/authorization"
	"github.com/FangcunMount/iam/v5/internal/apiserver/transport/rest/authz/handler"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterBindsAuthzRoutesToExplicitPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	captured := make(map[string]int)
	permission := func(resource, action string) gin.HandlerFunc {
		captured[resource+"/"+action]++
		return func(c *gin.Context) { c.Next() }
	}
	Register(gin.New(), Dependencies{
		RoleHandler:            handler.NewRoleHandler(nil, nil),
		AssignmentHandler:      handler.NewAssignmentHandler(nil, nil),
		PermissionGrantHandler: handler.NewPermissionGrantHandler(nil),
		ResourceHandler:        handler.NewResourceHandler(nil, nil),
		AuthMiddleware:         func(c *gin.Context) { c.Next() },
		Permission:             permission,
	})

	for _, key := range []string{
		authzapp.ResourcePermissionGrants + "/" + authzapp.ActionList,
		authzapp.ResourcePermissionGrants + "/" + authzapp.ActionCreate,
		authzapp.ResourcePermissionGrants + "/" + authzapp.ActionRevoke,
		authzapp.ResourceAssignments + "/" + authzapp.ActionList,
		authzapp.ResourceAssignments + "/" + authzapp.ActionGrant,
		authzapp.ResourceAssignments + "/" + authzapp.ActionRevoke,
	} {
		require.Positive(t, captured[key], "missing route permission %s", key)
	}
	for key := range captured {
		require.NotContains(t, key, "collection:policies")
		require.NotContains(t, key, "action:check")
	}
}

func TestRetiredInheritanceRoutesReturnGoneWithoutDependencies(t *testing.T) {
	engine := gin.New()
	Register(engine, Dependencies{})
	for _, tc := range []struct{ method, path string }{{"GET", "/api/v4/authz/role-inheritances"}, {"POST", "/api/v4/authz/role-inheritances"}, {"DELETE", "/api/v4/authz/role-inheritances/1"}} {
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, http.StatusGone, response.Code)
		require.Contains(t, response.Body.String(), "role_inheritance_retired")
	}
}
