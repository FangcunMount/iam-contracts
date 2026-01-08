package restful

import (
	"net/http"

	"github.com/gin-gonic/gin"

	appsuggest "github.com/FangcunMount/iam-contracts/internal/apiserver/application/suggest"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/suggest"
)

// Dependencies wires runtime dependencies for the handler.
type Dependencies struct {
	Service        *appsuggest.Service
	AuthMiddleware gin.HandlerFunc
}

var deps Dependencies

// Provide sets dependencies before Register is called.
func Provide(d Dependencies) {
	deps = d
}

// Register registers routes onto the engine.
func Register(engine *gin.Engine) {
	if engine == nil || deps.Service == nil {
		return
	}

	group := engine.Group("/api/v1/suggest")
	if deps.AuthMiddleware != nil {
		group.Use(deps.AuthMiddleware)
	}

	h := &Handler{svc: deps.Service}
	group.GET("/child", h.Child)
}

// Handler 提供 suggest 接口
type Handler struct {
	svc *appsuggest.Service
}

// Child 处理儿童联想查询
func (h *Handler) Child(c *gin.Context) {
	k := c.Query("k")
	if k == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing k"})
		return
	}
	list := h.svc.Suggest(c, k)
	if list == nil {
		list = []suggest.Term{}
	}
	c.JSON(http.StatusOK, list)
}
