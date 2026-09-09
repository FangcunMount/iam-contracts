package token

import (
	"fmt"
	"strings"
	"time"

	sessiondomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/session"
	"github.com/FangcunMount/iam/v5/internal/pkg/authnclaims"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// accessTokenClaimsFromSession 投影会话中已确定的身份、认证和签发上下文。
func accessTokenClaimsFromSession(sess *sessiondomain.Session) AccessTokenClaims {
	businessContext := sess.BusinessContext.Clone()
	authenticatedAt := sess.AuthContext.AuthenticatedAt
	if authenticatedAt.IsZero() {
		authenticatedAt = sess.CreatedAt
	}
	return AccessTokenClaims{
		UserID: sess.UserID, LoginIdentityID: sess.LoginIdentityID, SessionID: sess.SessionID,
		Subject: sess.UserID.String(), OrgID: businessContext.OrgID,
		AMR: sess.AuthContext.AMRStrings(), AuthenticatedAt: authenticatedAt, Attributes: businessContext.Attributes,
	}
}

func businessContextFromClaims(claims map[string]any) sessiondomain.BusinessContext {
	context := sessiondomain.BusinessContext{

		Attributes: authnclaims.EncodeJWTAttributes(claims),
	}
	if raw := businessOrgIDFromClaims(claims); raw != "" {
		if id, err := meta.ParseID(raw); err == nil {
			context.OrgID = id
		}
	}
	return context
}

func businessOrgIDFromClaims(claims map[string]any) string {
	return stringClaimValue(claims, "org_id")
}

func resolveAuthenticatedAt(claims map[string]any, fallback time.Time) time.Time {
	if value, ok := claims["auth_time"]; ok && value != nil {
		switch typed := value.(type) {
		case time.Time:
			if !typed.IsZero() {
				return typed.UTC()
			}
		case string:
			if text := strings.TrimSpace(typed); text != "" {
				if parsed, err := time.Parse(time.RFC3339, text); err == nil {
					return parsed.UTC()
				}
			}
		}
	}
	return fallback
}

func stringClaimValue(claims map[string]any, key string) string {
	if len(claims) == 0 {
		return ""
	}
	value, ok := claims[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
