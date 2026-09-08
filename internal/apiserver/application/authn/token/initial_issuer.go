package token

import (
	"context"
	perrors "github.com/FangcunMount/component-base/pkg/errors"
	sessiondomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/session"
	tokendomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/token"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

type RefreshTokenSaver interface {
	SaveRefreshToken(context.Context, *tokendomain.RefreshToken) error
}
type initialTokenIssuer struct {
	minter tokendomain.TokenSetMinter
	saver  RefreshTokenSaver
}

func NewInitialTokenIssuer(minter tokendomain.TokenSetMinter, saver RefreshTokenSaver) InitialTokenIssuer {
	return &initialTokenIssuer{minter: minter, saver: saver}
}

// IssueInitialTokens succeeds only after the initial refresh token is persisted.
// It neither creates nor revokes a session; the calling use case owns compensation.
func (s *initialTokenIssuer) IssueInitialTokens(ctx context.Context, sess *sessiondomain.Session) (*TokenPair, error) {
	if s.minter == nil || s.saver == nil {
		return nil, perrors.WithCode(code.ErrInternalServerError, "initial token issuer is not configured")
	}
	if sess == nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "session is required")
	}
	set, err := s.minter.MintTokenSet(ctx, sess)
	if err != nil {
		return nil, err
	}
	if set == nil || set.AccessToken == nil || set.RefreshToken == nil {
		return nil, perrors.WithCode(code.ErrInternalServerError, "token set minter returned incomplete token set")
	}
	if err := s.saver.SaveRefreshToken(ctx, set.RefreshToken); err != nil {
		return nil, perrors.WrapC(err, code.ErrInternalServerError, "failed to save refresh token")
	}
	return tokenPairFromDomain(set), nil
}
