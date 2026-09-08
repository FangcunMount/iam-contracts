package token

import (
	"context"
	"testing"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/stretchr/testify/require"

	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

type signatureVerifierStub struct {
	claims *AccessTokenClaims
}

func (s *signatureVerifierStub) VerifySignatureAndClaims(context.Context, string) (*AccessTokenClaims, error) {
	return s.claims, nil
}

type bearerTokenStoreStub struct {
	revoked map[string]bool
}

func (*bearerTokenStoreStub) SaveRefreshToken(context.Context, *RefreshToken) error { return nil }

func (*bearerTokenStoreStub) GetRefreshToken(context.Context, string) (*RefreshToken, error) {
	return nil, nil
}

func (*bearerTokenStoreStub) GetConsumedRefreshToken(context.Context, string) (*ConsumedRefreshToken, error) {
	return nil, nil
}

func (*bearerTokenStoreStub) RotateRefreshToken(context.Context, string, string, *RefreshToken) (bool, error) {
	return false, nil
}

func (*bearerTokenStoreStub) DeleteRefreshToken(context.Context, string) error { return nil }

func (s *bearerTokenStoreStub) MarkBearerTokenRevoked(_ context.Context, tokenID string, _ time.Duration) error {
	s.revoked[tokenID] = true
	return nil
}

func (s *bearerTokenStoreStub) IsBearerTokenRevoked(_ context.Context, tokenID string) (bool, error) {
	return s.revoked[tokenID], nil
}

type trackingSessionRevokerStub struct {
	revokeCalls int
}

func (s *trackingSessionRevokerStub) Revoke(context.Context, string, string, string) error {
	s.revokeCalls++
	return nil
}

func (*trackingSessionRevokerStub) RevokeByUser(context.Context, meta.ID, string, string) error {
	return nil
}

func (*trackingSessionRevokerStub) RevokeByLoginIdentity(context.Context, meta.ID, string, string) error {
	return nil
}

func TestAccessTokenRevocationPreservesSessionRevocation(t *testing.T) {
	claims := &AccessTokenClaims{TokenID: "access-id", TokenType: TokenTypeAccess, Audience: []string{"qs-api"}, SessionID: "sid", Subject: "1", ExpiresAt: time.Now().Add(time.Hour)}
	codec := &signatureVerifierStub{claims: claims}
	store := &bearerTokenStoreStub{revoked: map[string]bool{}}
	sessions := &trackingSessionRevokerStub{}
	require.NoError(t, newRevoker(codec, store, sessions).RevokeBearerToken(context.Background(), "access"))
	require.True(t, store.revoked[claims.TokenID])
	require.Equal(t, 1, sessions.revokeCalls)
	_, err := newVerifier(codec, store, nil, nil).VerifyToken(context.Background(), "access", []string{"qs-api"})
	require.Equal(t, code.ErrTokenInvalid, perrors.ParseCoder(err).Code())
}

func TestRetiredTypeRejectedBeforeSessionAndStoreAccess(t *testing.T) {
	codec := &signatureVerifierStub{claims: &AccessTokenClaims{TokenType: TokenType("service")}}
	_, err := newVerifier(codec, nil, nil, nil).VerifyToken(context.Background(), "retired", []string{"qs-api"})
	require.Equal(t, code.ErrTokenInvalid, perrors.ParseCoder(err).Code())
}
