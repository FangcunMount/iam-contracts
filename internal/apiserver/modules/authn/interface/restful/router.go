package restful

import (
	"net/http"

	"github.com/gin-gonic/gin"

	authhandler "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/interface/restful/handler"
	authstrategys "github.com/fangcun-mount/iam-contracts/internal/pkg/middleware/auth/strategys"
)

// Dependencies describes the external collaborators needed to expose authn endpoints.
type Dependencies struct {
	JWTStrategy    *authstrategys.JWTStrategy
	AccountHandler *authhandler.AccountHandler
}

var deps Dependencies

// Provide wires the dependencies for subsequent Register calls.
func Provide(d Dependencies) {
	deps = d
}

// Register exposes the authentication endpoints that issue and refresh tokens.
func Register(engine *gin.Engine) {
	if engine == nil {
		return
	}

	api := engine.Group("/api")
	registerAuthEndpoints(api.Group("/v1/auth"), deps.JWTStrategy)
	registerAccountEndpoints(api.Group("/v1"), deps.AccountHandler)
}

func registerAuthEndpoints(group *gin.RouterGroup, strategy *authstrategys.JWTStrategy) {
	if group == nil || strategy == nil {
		return
	}
	group.POST("/login", strategy.LoginHandler)
	group.POST("/logout", strategy.LogoutHandler)
	group.POST("/token", strategy.RefreshHandler)
	group.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

func registerAccountEndpoints(v1 *gin.RouterGroup, h *authhandler.AccountHandler) {
	if v1 == nil || h == nil {
		return
	}

	accounts := v1.Group("/accounts")
	accounts.POST("/operation", h.CreateOperationAccount)
	accounts.PATCH("/operation/:username", h.UpdateOperationCredential)
	accounts.POST("/operation/:username:change", h.ChangeOperationUsername)
	accounts.POST("/wechat:bind", h.BindWeChatAccount)
	accounts.PATCH("/:accountId/wechat:profile", h.UpsertWeChatProfile)
	accounts.PATCH("/:accountId/wechat:unionid", h.SetWeChatUnionID)
	accounts.GET("/:accountId", h.GetAccount)
	accounts.POST("/:accountId:enable", h.EnableAccount)
	accounts.POST("/:accountId:disable", h.DisableAccount)
	accounts.GET("/operation/:username", h.GetOperationAccountByUsername)

	v1.GET("/accounts:by-ref", h.FindAccountByRef)

	users := v1.Group("/users")
	users.GET("/:userId/accounts", h.ListAccountsByUser)
}
