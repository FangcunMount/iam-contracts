package suggest

import (
	"github.com/gin-gonic/gin"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/suggest/visibility"
	"github.com/FangcunMount/iam/v5/internal/pkg/requestctx"
)

// OperatingPrincipalFromGin 从 JWT 上下文提取 suggest 用身份快照。
func OperatingPrincipalFromGin(c *gin.Context) (visibility.Principal, bool) {
	if c == nil {
		return visibility.Principal{}, false
	}
	uid, ok := requestctx.UserID(c)
	if !ok || uid.IsZero() {
		return visibility.Principal{}, false
	}
	principal := visibility.Principal{
		OperatorID: int64(uid),
	}
	if orgID, ok := requestctx.BusinessOrgID(c); ok {
		principal.OrgID = int64(orgID)
	}
	return principal, true
}
