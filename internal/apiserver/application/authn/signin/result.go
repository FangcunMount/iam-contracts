package signin

import (
	tokenapp "github.com/FangcunMount/iam/v4/internal/apiserver/application/authn/token"
	"github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/authentication"
	sessiondomain "github.com/FangcunMount/iam/v4/internal/apiserver/domain/authn/session"
	"github.com/FangcunMount/iam/v4/internal/pkg/meta"
)

// Result 是登录成功后的应用层结果。
type Result struct {
	Principal       *authentication.Principal
	TokenPair       *tokenapp.TokenPair
	UserID          meta.ID
	LoginIdentityID meta.ID
}

// resultFromSession 在身份与会话已对齐且令牌颁发完成后构造登录结果。
func resultFromSession(principal *authentication.Principal, sess *sessiondomain.Session, tokenPair *tokenapp.TokenPair) *Result {
	// 如果认证主体为空，返回仅包含令牌对的登录结果
	if principal == nil {
		return &Result{TokenPair: tokenPair}
	}
	return &Result{
		Principal:       principal,
		TokenPair:       tokenPair,
		UserID:          sess.UserID,
		LoginIdentityID: sess.LoginIdentityID,
	}
}
