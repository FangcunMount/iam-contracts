package token

import (
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/authentication"
	sessiondomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/session"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestAccessTokenClaimsProjectionUsesSessionContext(t *testing.T) {
	sess := &sessiondomain.Session{
		SessionID: "sid-1", UserID: meta.FromUint64(10), LoginIdentityID: meta.FromUint64(20),
		AuthContext:     authentication.RestoreAuthenticationContext(authentication.MethodPassword, "wx-app", []authentication.AMR{authentication.AMRPassword}, time.Time{}),
		BusinessContext: sessiondomain.BusinessContext{OrgID: meta.FromUint64(42), Attributes: map[string]string{"key": "value"}},
	}
	got := accessTokenClaimsFromSession(sess)
	require.Equal(t, "sid-1", got.SessionID)
	require.Equal(t, sess.UserID, got.UserID)
	require.Equal(t, sess.LoginIdentityID, got.LoginIdentityID)
	require.Equal(t, meta.FromUint64(42), got.OrgID)
	require.Equal(t, []string{"pwd"}, got.AMR)
	got.Attributes["key"] = "changed"
	require.Equal(t, "value", sess.BusinessContext.Attributes["key"])
}
func TestAccessTokenClaimsProjectionDoesNotUseRealmAsOrg(t *testing.T) {
	sess := &sessiondomain.Session{
		AuthContext: authentication.RestoreAuthenticationContext(authentication.MethodWechatMinip, "wx-app-id", nil, time.Time{}),
	}
	got := accessTokenClaimsFromSession(sess)
	require.True(t, got.OrgID.IsZero())
}
