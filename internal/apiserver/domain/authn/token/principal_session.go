package token

import (
	"fmt"
	"strings"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/authentication"
	sessiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
	"github.com/FangcunMount/iam/v4/internal/pkg/authnclaims"
	"github.com/FangcunMount/iam/v4/internal/pkg/code"
	"github.com/FangcunMount/iam/v4/internal/pkg/meta"
	"github.com/FangcunMount/iam/v4/pkg/tenant"
)

// validatePrincipalSessionAlignment 验证 principal 和 session 是否对齐
func validatePrincipalSessionAlignment(principal *authentication.Principal, sess *sessiondomain.Session) error {
	if principal == nil || sess == nil {
		return perrors.WithCode(code.ErrInvalidArgument, "principal and session are required")
	}
	if principal.UserID != sess.UserID {
		return perrors.WithCode(code.ErrInvalidArgument, "principal user does not match session")
	}
	if principal.LoginIdentityID != sess.LoginIdentityID {
		return perrors.WithCode(code.ErrInvalidArgument, "principal login identity does not match session")
	}
	if sessionID := strings.TrimSpace(principal.SessionID); sessionID != "" && sessionID != sess.SessionID {
		return perrors.WithCode(code.ErrInvalidArgument, "principal session id does not match session")
	}
	if !principal.TenantID.IsZero() && !sess.TenantID.IsZero() && principal.TenantID != sess.TenantID {
		return perrors.WithCode(code.ErrInvalidArgument, "principal tenant does not match session")
	}
	if principal.AuthContext.Method != "" && sess.AuthContext.Method != "" && principal.AuthContext.Method != sess.AuthContext.Method {
		return perrors.WithCode(code.ErrInvalidArgument, "principal auth method does not match session")
	}
	if principal.AuthContext.Realm != "" && sess.AuthContext.Realm != "" && principal.AuthContext.Realm != sess.AuthContext.Realm {
		return perrors.WithCode(code.ErrInvalidArgument, "principal realm does not match session")
	}
	return nil
}

// resolveMintTenantID 解析颁发令牌的租户ID
func resolveMintTenantID(principal *authentication.Principal, sess *sessiondomain.Session) meta.ID {
	if sess != nil && !sess.TenantID.IsZero() {
		return sess.TenantID
	}
	if principal != nil && !principal.TenantID.IsZero() {
		return principal.TenantID
	}
	return meta.ZeroID
}

// accessTokenSubjectFromAuth 从认证结果中生成访问令牌主体。
// 授权域只来自显式 TokenContext，不再用 Realm 兜底。
func accessTokenSubjectFromAuth(principal *authentication.Principal, sess *sessiondomain.Session) *AccessTokenSubject {
	// 克隆认证上下文
	tokenContext := principal.TokenContext.Clone()
	// 如果会话存在，则使用会话的认证上下文
	if sess != nil {
		if tokenContext.TenantDomain == "" {
			tokenContext.TenantDomain = sess.TokenContext.TenantDomain
		}
		// 如果授权域ID为空，则使用会话的授权域ID
		if tokenContext.OrgID.IsZero() {
			tokenContext.OrgID = sess.TokenContext.OrgID
		}
		// 如果属性为空，则使用会话的属性
		if len(tokenContext.Attributes) == 0 {
			tokenContext.Attributes = sess.TokenContext.Clone().Attributes
		}
	}
	// 如果授权域域名为空，则使用默认域名
	if tokenContext.TenantDomain == "" {
		tokenContext.TenantDomain = tenant.DefaultID
	}
	// 获取认证时间
	authenticatedAt := principal.AuthContext.AuthenticatedAt
	// 获取会话ID
	sessionID := ""
	if sess != nil {
		sessionID = sess.SessionID
		// 如果认证时间为空，则使用会话的认证时间
		if authenticatedAt.IsZero() {
			authenticatedAt = sess.AuthContext.AuthenticatedAt
		}
		// 如果认证时间为空，则使用会话的创建时间
		if authenticatedAt.IsZero() {
			authenticatedAt = sess.CreatedAt
		}
	}
	// 获取认证方法
	amr := principal.AuthContext.AMRStrings()
	// 如果认证方法为空，则使用会话的认证方法
	if len(amr) == 0 && sess != nil {
		amr = sess.AuthContext.AMRStrings()
	}
	// 获取授权域ID
	orgID := ""
	if !tokenContext.OrgID.IsZero() {
		orgID = tokenContext.OrgID.String()
	}

	// 返回访问令牌主体
	return &AccessTokenSubject{
		UserID:          principal.UserID,
		LoginIdentityID: principal.LoginIdentityID,
		SessionID:       sessionID,
		TenantID:        resolveMintTenantID(principal, sess),
		TenantDomain:    tokenContext.TenantDomain,
		OrgID:           orgID,
		AMR:             amr,
		AuthenticatedAt: authenticatedAt,
		Attributes:      cloneStringMap(tokenContext.Attributes),
	}
}

func tokenContextFromClaims(claims map[string]any) authentication.TokenContext {
	context := authentication.TokenContext{
		TenantDomain: resolveTenantDomain(claims),
		Attributes:   authnclaims.EncodeJWTAttributes(claims),
	}
	if raw := businessOrgIDFromClaims(claims); raw != "" {
		if id, err := meta.ParseID(raw); err == nil {
			context.OrgID = id
		}
	}
	return context
}

func resolveTenantDomain(claims map[string]any) string {
	if domain := stringClaimValue(claims, "tenant_domain"); domain != "" {
		return domain
	}
	return tenant.DefaultID
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
