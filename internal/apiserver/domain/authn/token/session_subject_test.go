package token

import (
	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/authentication"
	sessiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
	"github.com/FangcunMount/iam/v4/internal/pkg/meta"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAccessTokenClaimsProjectionUsesSessionContext(t *testing.T) {
	sess := &sessiondomain.Session{
		SessionID: "sid-1", UserID: meta.FromUint64(10), LoginIdentityID: meta.FromUint64(20),
		AuthContext:  authentication.RestoreAuthenticationContext(authentication.MethodPassword, "wx-app", []authentication.AMR{authentication.AMRPassword}, time.Time{}),
		TokenContext: sessiondomain.TokenContext{TenantDomain: "fangcun", OrgID: meta.FromUint64(42), Attributes: map[string]string{"key": "value"}},
	}
	got := accessTokenClaimsFromSession(sess)
	require.Equal(t, "sid-1", got.SessionID)
	require.Equal(t, sess.UserID, got.UserID)
	require.Equal(t, sess.LoginIdentityID, got.LoginIdentityID)
	require.Equal(t, "fangcun", got.TenantDomain)
	require.Equal(t, meta.FromUint64(42), got.OrgID)
	require.Equal(t, []string{"pwd"}, got.AMR)
	got.Attributes["key"] = "changed"
	require.Equal(t, "value", sess.TokenContext.Attributes["key"])
}
func TestAccessTokenClaimsProjectionDoesNotUseRealmAsTenantDomain(t *testing.T) {
	sess := &sessiondomain.Session{
		AuthContext: authentication.RestoreAuthenticationContext(authentication.MethodWechatMinip, "wx-app-id", nil, time.Time{}),
	}
	got := accessTokenClaimsFromSession(sess)
	require.Equal(t, "fangcun", got.TenantDomain)
}
