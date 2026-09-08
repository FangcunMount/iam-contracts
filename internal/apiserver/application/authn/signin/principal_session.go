package signin

import (
	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/authentication"
	sessiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
	"github.com/FangcunMount/iam/v4/internal/pkg/code"
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
	if principal.AuthContext.Method != "" && sess.AuthContext.Method != "" && principal.AuthContext.Method != sess.AuthContext.Method {
		return perrors.WithCode(code.ErrInvalidArgument, "principal auth method does not match session")
	}
	if principal.AuthContext.Realm != "" && sess.AuthContext.Realm != "" && principal.AuthContext.Realm != sess.AuthContext.Realm {
		return perrors.WithCode(code.ErrInvalidArgument, "principal realm does not match session")
	}
	return nil
}
