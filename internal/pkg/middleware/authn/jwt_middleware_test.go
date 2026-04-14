package authn

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	tokenDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/token"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

func TestApplyVerifiedClaimsSetsTenantIDForRoleResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/identity/me", nil)

	claims := tokenDomain.NewTokenClaims(
		tokenDomain.TokenTypeAccess,
		"token-1",
		"user:110001",
		meta.ID(110001),
		meta.ID(613486856213901870),
		meta.ID(1),
		"https://iam.fangcunmount.cn",
		[]string{"qs-api"},
		nil,
		[]string{"pwd"},
		time.Now(),
		time.Now().Add(time.Hour),
	)

	applyVerifiedClaims(c, claims)

	if got := TenantIDFromGin(c); got != "1" {
		t.Fatalf("TenantIDFromGin() = %q, want %q", got, "1")
	}
	if got, exists := c.Get(ContextKeyUserID); !exists || got != "110001" {
		t.Fatalf("gin user_id = %v exists=%v, want %q", got, exists, "110001")
	}
	if got, exists := c.Get(ContextKeyAccountID); !exists || got != "613486856213901870" {
		t.Fatalf("gin account_id = %v exists=%v, want %q", got, exists, "613486856213901870")
	}
	if got, exists := c.Get(ContextKeyTokenID); !exists || got != "token-1" {
		t.Fatalf("gin token_id = %v exists=%v, want %q", got, exists, "token-1")
	}
}
