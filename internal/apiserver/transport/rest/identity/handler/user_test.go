package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"

	appuser "github.com/FangcunMount/iam/v5/internal/apiserver/application/identity/user"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

func TestResolveRolesUsesUnifiedSpace(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v2/identity/me", nil)

	h := &UserHandler{effectiveRoles: userRoleLookupStub{roles: []string{"qs:admin", "platform_admin"}}}

	got := h.resolveRoles(c, meta.FromUint64(10001))
	want := []string{"qs:admin", "platform_admin"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveRoles() = %#v, want %#v", got, want)
	}
}

func TestNewUserResponseUsesNicknameAndFallsBackToName(t *testing.T) {
	got := newUserResponse(&appuser.UserResult{
		ID:       "10001",
		Name:     "法定名",
		Nickname: "昵称",
	}, nil)
	if got.Nickname != "昵称" {
		t.Fatalf("nickname = %q, want %q", got.Nickname, "昵称")
	}

	got = newUserResponse(&appuser.UserResult{
		ID:   "10001",
		Name: "展示名",
	}, nil)
	if got.Nickname != "展示名" {
		t.Fatalf("fallback nickname = %q, want %q", got.Nickname, "展示名")
	}
}

type userRoleLookupStub struct {
	roles []string
}

func (s userRoleLookupStub) EffectiveRoleNamesForSubject(_ context.Context, _ subject.Ref) ([]string, error) {
	return append([]string(nil), s.roles...), nil
}
