package token_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/token"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

type genStub struct {
	tok *token.Token
	err error
}

func (g *genStub) GenerateAccessToken(pr *authentication.Principal, expiresIn time.Duration) (*token.Token, error) {
	if g.err != nil {
		return nil, g.err
	}
	return g.tok, nil
}
func (g *genStub) ParseAccessToken(tokenValue string) (*token.TokenClaims, error) {
	if g.tok == nil {
		return nil, errors.New("no token")
	}
	// return claims built from g.tok
	return token.NewTokenClaims(g.tok.ID, g.tok.UserID, g.tok.AccountID, g.tok.IssuedAt, g.tok.ExpiresAt), nil
}

type storeStub struct {
	saved             *token.Token
	blacklistedID     string
	blacklistedExpiry time.Duration
	saveErr           error
	blacklistErr      error
	addCalled         int
}

func (s *storeStub) SaveRefreshToken(ctx context.Context, t *token.Token) error {
	s.saved = t
	return s.saveErr
}
func (s *storeStub) GetRefreshToken(ctx context.Context, tokenValue string) (*token.Token, error) {
	return nil, nil
}
func (s *storeStub) DeleteRefreshToken(ctx context.Context, tokenValue string) error { return nil }
func (s *storeStub) AddToBlacklist(ctx context.Context, tokenID string, expiry time.Duration) error {
	s.addCalled++
	s.blacklistedID = tokenID
	s.blacklistedExpiry = expiry
	return s.blacklistErr
}
func (s *storeStub) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	return false, nil
}

func TestIssueToken_HappyPathAndGeneratorError(t *testing.T) {
	acc := &authentication.Principal{AccountID: meta.FromUint64(2), UserID: meta.FromUint64(1), TenantID: meta.FromUint64(0)}

	// happy path
	access := token.NewAccessToken("aid", "aval", acc.UserID, acc.AccountID, time.Minute)
	gen := &genStub{tok: access}
	store := &storeStub{}
	issuer := token.NewTokenIssuer(gen, store, time.Minute, time.Hour)

	pair, err := issuer.IssueToken(context.Background(), acc)
	require.NoError(t, err)
	require.NotNil(t, pair)
	require.Equal(t, access.Value, pair.AccessToken.Value)
	require.NotNil(t, store.saved)
	require.Equal(t, pair.RefreshToken.UserID, acc.UserID)

	// generator error
	genErr := &genStub{err: errors.New("gen fail")}
	issuer2 := token.NewTokenIssuer(genErr, store, time.Minute, time.Hour)
	_, err2 := issuer2.IssueToken(context.Background(), acc)
	require.Error(t, err2)
}

func TestRevokeToken_ExpiredAndBlacklist(t *testing.T) {
	// expired token: Parse returns claims with past expiry
	expired := token.NewAccessToken("eid", "eval", meta.FromUint64(1), meta.FromUint64(2), -time.Minute)
	gen := &genStub{tok: expired}
	store := &storeStub{}
	issuer := token.NewTokenIssuer(gen, store, time.Minute, time.Hour)

	// Revoke should return nil when token already expired (no blacklist)
	err := issuer.RevokeToken(context.Background(), expired.Value)
	require.NoError(t, err)
	require.Equal(t, 0, store.addCalled)

	// future token -> blacklist called
	futureTok := token.NewAccessToken("fid", "fval", meta.FromUint64(9), meta.FromUint64(8), time.Hour)
	gen2 := &genStub{tok: futureTok}
	store2 := &storeStub{}
	issuer2 := token.NewTokenIssuer(gen2, store2, time.Minute, time.Hour)
	err2 := issuer2.RevokeToken(context.Background(), futureTok.Value)
	require.NoError(t, err2)
	require.Equal(t, 1, store2.addCalled)
	require.Equal(t, futureTok.ID, store2.blacklistedID)
	// expiry should be close to RemainingDuration
	rem := futureTok.RemainingDuration()
	// allow small delta
	require.True(t, store2.blacklistedExpiry <= rem+time.Second && store2.blacklistedExpiry >= rem-time.Second)
}
