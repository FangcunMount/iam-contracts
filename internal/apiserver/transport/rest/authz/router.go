package authz

import (
	"net/http"

	authzapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/authorization"
	"github.com/FangcunMount/iam/v5/internal/apiserver/transport/rest/authz/handler"
	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	RoleHandler            *handler.RoleHandler
	AssignmentHandler      *handler.AssignmentHandler
	PermissionGrantHandler *handler.PermissionGrantHandler
	ResourceHandler        *handler.ResourceHandler
	AuthMiddleware         gin.HandlerFunc
	Permission             func(resource, action string) gin.HandlerFunc
}

func Register(engine *gin.Engine, deps Dependencies) {
	if engine == nil {
		return
	}
	authzGroup := engine.Group("/api/v4/authz")
	authzGroup.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "module": "authz"})
	})
	// A tombstone route deliberately has no authentication, authorization or
	// database dependency: the capability no longer exists.
	authzGroup.GET("/role-inheritances", retiredInheritance)
	authzGroup.POST("/role-inheritances", retiredInheritance)
	authzGroup.DELETE("/role-inheritances/:id", retiredInheritance)
	if deps.RoleHandler == nil || deps.AuthMiddleware == nil || deps.Permission == nil {
		return
	}
	g := authzGroup.Group("")
	g.Use(deps.AuthMiddleware)

	roles := g.Group("/roles")
	roles.POST("", deps.Permission(authzapp.ResourceRoles, authzapp.ActionCreate), deps.RoleHandler.CreateRole)
	roles.PUT("/:id", deps.Permission(authzapp.ResourceRoles, authzapp.ActionUpdate), deps.RoleHandler.UpdateRole)
	roles.DELETE("/:id", deps.Permission(authzapp.ResourceRoles, authzapp.ActionDelete), deps.RoleHandler.DeleteRole)
	roles.GET("/:id", deps.Permission(authzapp.ResourceRoles, authzapp.ActionRead), deps.RoleHandler.GetRole)
	roles.GET("", deps.Permission(authzapp.ResourceRoles, authzapp.ActionList), deps.RoleHandler.ListRoles)
	if deps.AssignmentHandler != nil {
		roles.GET("/:id/assignments", deps.Permission(authzapp.ResourceAssignments, authzapp.ActionList), deps.AssignmentHandler.ListAssignmentsByRole)
	}
	if deps.PermissionGrantHandler != nil {
		roles.GET("/:id/grants", deps.Permission(authzapp.ResourcePermissionGrants, authzapp.ActionList), deps.PermissionGrantHandler.ListRoleGrants)
		g.POST("/grants", deps.Permission(authzapp.ResourcePermissionGrants, authzapp.ActionCreate), deps.PermissionGrantHandler.CreateGrant)
		g.DELETE("/grants/:id", deps.Permission(authzapp.ResourcePermissionGrants, authzapp.ActionRevoke), deps.PermissionGrantHandler.RevokeGrant)
	}

	if deps.AssignmentHandler != nil {
		assignments := g.Group("/assignments")
		assignments.POST("/grant", deps.Permission(authzapp.ResourceAssignments, authzapp.ActionGrant), deps.AssignmentHandler.GrantAssignment)
		assignments.POST("/revoke", deps.Permission(authzapp.ResourceAssignments, authzapp.ActionRevoke), deps.AssignmentHandler.RevokeAssignment)
		assignments.DELETE("/:id", deps.Permission(authzapp.ResourceAssignments, authzapp.ActionRevoke), deps.AssignmentHandler.RevokeAssignmentByID)
		assignments.GET("/subject", deps.Permission(authzapp.ResourceAssignments, authzapp.ActionList), deps.AssignmentHandler.ListAssignmentsBySubject)
	}

	if deps.ResourceHandler != nil {
		resources := g.Group("/resources")
		resources.POST("", deps.Permission(authzapp.ResourceResources, authzapp.ActionCreate), deps.ResourceHandler.CreateResource)
		resources.PUT("/:id", deps.Permission(authzapp.ResourceResources, authzapp.ActionUpdate), deps.ResourceHandler.UpdateResource)
		resources.DELETE("/:id", deps.Permission(authzapp.ResourceResources, authzapp.ActionDelete), deps.ResourceHandler.DeleteResource)
		resources.GET("/:id", deps.Permission(authzapp.ResourceResources, authzapp.ActionRead), deps.ResourceHandler.GetResource)
		resources.GET("/key/:key", deps.Permission(authzapp.ResourceResources, authzapp.ActionRead), deps.ResourceHandler.GetResourceByKey)
		resources.GET("", deps.Permission(authzapp.ResourceResources, authzapp.ActionList), deps.ResourceHandler.ListResources)
		resources.POST("/validate-action", deps.Permission(authzapp.ResourceResources, authzapp.ActionValidateAction), deps.ResourceHandler.ValidateAction)
	}
}

// retiredInheritance reports the retired capability without database access.
// @Summary 角色继承已退役
// @Tags AuthZ
// @Produce json
// @Failure 410 {object} map[string]string "角色继承已退役，请直接分配多个角色"
// @Router /v4/authz/role-inheritances [get]
// @Router /v4/authz/role-inheritances [post]
// @Router /v4/authz/role-inheritances/{id} [delete]
func retiredInheritance(c *gin.Context) {
	c.JSON(http.StatusGone, gin.H{"code": "role_inheritance_retired", "message": "角色继承已退役，请直接分配多个角色"})
}
