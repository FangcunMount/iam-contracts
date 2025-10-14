package restful

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Dependencies is kept for future authorization collaborators.
type Dependencies struct{}

var deps Dependencies

// Provide stores the dependencies for Register; kept for API symmetry.
func Provide(d Dependencies) {
	deps = d
}

// Register exposes placeholder authorization routes.
func Register(engine *gin.Engine) {
	if engine == nil {
		return
	}

	group := engine.Group("/authz")
	{
		group.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
			})
		})
	}
}
