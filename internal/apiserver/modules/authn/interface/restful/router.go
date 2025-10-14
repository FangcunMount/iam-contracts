package restful

import (
	"net/http"

	"github.com/gin-gonic/gin"

	authstrategys "github.com/fangcun-mount/iam-contracts/internal/pkg/middleware/auth/strategys"
)

// Dependencies describes the external collaborators needed to expose authn endpoints.
type Dependencies struct {
	JWTStrategy *authstrategys.JWTStrategy
}

var deps Dependencies

// Provide wires the dependencies for subsequent Register calls.
func Provide(d Dependencies) {
	deps = d
}

// Register exposes the authentication endpoints that issue and refresh tokens.
func Register(engine *gin.Engine) {
	if engine == nil || deps.JWTStrategy == nil {
		return
	}

	auth := engine.Group("/auth")
	{
		auth.POST("/login", deps.JWTStrategy.LoginHandler)
		auth.POST("/logout", deps.JWTStrategy.LogoutHandler)
		auth.POST("/refresh", deps.JWTStrategy.RefreshHandler)
		auth.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}
}
