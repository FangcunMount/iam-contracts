package token

import (
	"testing"
	"time"

	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/authentication"
	sessiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
	"github.com/FangcunMount/iam/v4/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestSessionForRefreshPrefersSessionContext(t *testing.T) {
	s := &refresher{legacyContextDecoder: normalizeLegacyContextDecoder(nil)}
	authenticatedAt := time.Unix(1700000100, 0).UTC()
	sess := sessiondomain.NewWithContexts(
		"sid", meta.FromUint64(1), meta.FromUint64(2),
		authentication.RestoreAuthenticationContext(authentication.MethodPassword, "global", []authentication.AMR{authentication.AMRPassword}, authenticatedAt),
		sessiondomain.TokenContext{TenantDomain: "fangcun"}, time.Now().Add(time.Hour),
	)

	refresh := RestoreRefreshToken("rid", "rval", "sid", meta.FromUint64(1), meta.FromUint64(2), time.Now().Add(time.Hour), LegacyRefreshContext{AMR: []string{"otp"}, SessionClaims: map[string]string{"tenant_domain": "legacy"}})
	refresh.AuthMethod = "phone_otp"
	refresh.Realm = "legacy-realm"

	restored := s.sessionForRefresh(sess, refresh)
	require.Equal(t, "password", string(restored.AuthContext.Method))
	require.Equal(t, "global", restored.AuthContext.Realm)
	require.Equal(t, []string{"pwd"}, restored.AuthContext.AMRStrings())
	require.Equal(t, "fangcun", restored.TokenContext.TenantDomain)
	require.Equal(t, authenticatedAt, restored.AuthContext.AuthenticatedAt)
}

func TestSessionForRefreshFallsBackToRefreshToken(t *testing.T) {
	s := &refresher{legacyContextDecoder: normalizeLegacyContextDecoder(nil)}
	sess := &sessiondomain.Session{
		SessionID: "sid", UserID: meta.FromUint64(1), LoginIdentityID: meta.FromUint64(2),
		Status: sessiondomain.StatusActive, CreatedAt: time.Now().Add(-time.Hour), ExpiresAt: time.Now().Add(time.Hour),
	}
	refresh := RestoreRefreshToken("rid", "rval", "sid", meta.FromUint64(1), meta.FromUint64(2), time.Now().Add(time.Hour), LegacyRefreshContext{AMR: []string{"otp"}, SessionClaims: map[string]string{"tenant_domain": "legacy"}})
	refresh.AuthMethod = "phone_otp"
	refresh.Realm = "legacy-realm"

	restored := s.sessionForRefresh(sess, refresh)
	require.Equal(t, "phone_otp", string(restored.AuthContext.Method))
	require.Equal(t, "legacy-realm", restored.AuthContext.Realm)
	require.Equal(t, []string{"otp"}, restored.AuthContext.AMRStrings())
	require.Equal(t, "legacy", restored.TokenContext.TenantDomain)
	require.Equal(t, sess.CreatedAt.UTC(), restored.AuthContext.AuthenticatedAt)
}

func TestSessionForRefreshDoesNotInventMissingHistoricalAuthTime(t *testing.T) {
	s := &refresher{legacyContextDecoder: normalizeLegacyContextDecoder(nil)}
	sess := &sessiondomain.Session{
		SessionID: "sid", UserID: meta.FromUint64(1), LoginIdentityID: meta.FromUint64(2),
		Status: sessiondomain.StatusActive, ExpiresAt: time.Now().Add(time.Hour),
	}
	refresh := RestoreRefreshToken("rid", "rval", "sid", meta.FromUint64(1), meta.FromUint64(2), time.Now().Add(time.Hour), LegacyRefreshContext{AMR: []string{"pwd"}, SessionClaims: nil})
	refresh.AuthMethod = "password"
	refresh.Realm = "global"

	restored := s.sessionForRefresh(sess, refresh)
	require.True(t, restored.AuthContext.AuthenticatedAt.IsZero())
}

func TestAccessTokenClaimsProjectionKeepsAuthContextAuthenticatedAt(t *testing.T) {
	authenticatedAt := time.Unix(1700000200, 0).UTC()
	sess := &sessiondomain.Session{
		SessionID: "sid-1", UserID: meta.FromUint64(10), LoginIdentityID: meta.FromUint64(20),
		TokenContext: sessiondomain.TokenContext{TenantDomain: "fangcun"},
		AuthContext:  authentication.NewAuthenticationContext(authentication.MethodPassword, "global", []authentication.AMR{authentication.AMRPassword}, authenticatedAt),
		CreatedAt:    time.Unix(1, 0).UTC(),
	}
	got := accessTokenClaimsFromSession(sess)
	require.Equal(t, authenticatedAt, got.AuthenticatedAt)
}

func TestSessionForRefreshRestoresLegacyContextWithoutMutatingLoadedSession(t *testing.T) {
	s := &refresher{legacyContextDecoder: normalizeLegacyContextDecoder(nil)}
	sess := &sessiondomain.Session{SessionID: "sid", CreatedAt: time.Unix(1700000000, 0).UTC()}
	refresh := RestoreRefreshToken("rid", "value", "sid", meta.FromUint64(1), meta.FromUint64(2), time.Now().Add(time.Hour), LegacyRefreshContext{AMR: []string{"pwd"}, SessionClaims: map[string]string{"tenant_domain": "legacy"}})
	refresh.AuthMethod = "password"
	refresh.Realm = "global"
	restored := s.sessionForRefresh(sess, refresh)
	require.NotSame(t, sess, restored)
	require.Empty(t, sess.AuthContext.Method)
	require.Empty(t, sess.TokenContext.TenantDomain)
	subject := accessTokenClaimsFromSession(restored)
	require.Equal(t, "sid", subject.SessionID)
	require.Equal(t, "legacy", subject.TenantDomain)
	require.Equal(t, []string{"pwd"}, subject.AMR)
	require.Equal(t, sess.CreatedAt, subject.AuthenticatedAt)
}
