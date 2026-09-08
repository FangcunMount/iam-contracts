package verifier

import (
	"context"
	"testing"
	"time"

	authnv2 "github.com/FangcunMount/iam/v4/api/grpc/iam/authn/v2"
	iamerrors "github.com/FangcunMount/iam/v4/pkg/sdk/errors"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/require"
)

func TestRetiredAndUnknownTypesCannotBeOptedInto(t *testing.T) {
	key, manager := newRS256Fixture(t)
	for _, kind := range []string{"service", "unknown", "refresh"} {
		t.Run(kind, func(t *testing.T) {
			token := signRS256Token(t, key, map[string]interface{}{jwt.IssuerKey: "https://iam.fangcunmount.cn", jwt.AudienceKey: []string{"qs-api"}, jwt.SubjectKey: "1", jwt.ExpirationKey: time.Now().Add(time.Minute), "token_type": kind})
			opts := &VerifyOptions{AllowedTokenTypes: []authnv2.TokenType{authnv2.TokenType(3), authnv2.TokenType_TOKEN_TYPE_UNSPECIFIED, authnv2.TokenType_TOKEN_TYPE_REFRESH}}
			_, err := NewLocalVerifyStrategy(manager, WithLocalConfig(testRecipientConfig())).Verify(context.Background(), token, opts)
			require.ErrorIs(t, err, iamerrors.ErrTokenInvalid)
			remote := NewRemoteVerifyStrategy(&verifyTokenClientStub{verifyResp: &authnv2.VerifyTokenResponse{Valid: true, Claims: &authnv2.TokenClaims{TokenType: authnv2.TokenType_TOKEN_TYPE_ACCESS}}}, testRecipientConfig())
			_, err = remote.Verify(context.Background(), token, opts)
			require.ErrorIs(t, err, iamerrors.ErrTokenInvalid)
		})
	}
}

func TestRemoteUnknownTypesDoNotBecomeLegacyAccess(t *testing.T) {
	key, _ := newRS256Fixture(t)
	token := signRS256Token(t, key, map[string]interface{}{jwt.IssuerKey: "https://iam.fangcunmount.cn", jwt.AudienceKey: []string{"qs-api"}, jwt.SubjectKey: "1", jwt.ExpirationKey: time.Now().Add(time.Minute)})
	for _, kind := range []authnv2.TokenType{0, 1, 3, 99} {
		stub := &verifyTokenClientStub{verifyResp: &authnv2.VerifyTokenResponse{Valid: true, Claims: &authnv2.TokenClaims{TokenType: kind}}}
		result, err := NewRemoteVerifyStrategy(stub, testRecipientConfig()).Verify(context.Background(), token, nil)
		if kind == 0 || kind == 1 {
			require.NoError(t, err)
			require.Equal(t, "access", result.Claims.TokenType)
		} else {
			require.ErrorIs(t, err, iamerrors.ErrTokenInvalid)
			require.Nil(t, result)
		}
	}
	stub := &verifyTokenClientStub{verifyResp: &authnv2.VerifyTokenResponse{Valid: true, Claims: &authnv2.TokenClaims{TokenType: 1}, Metadata: &authnv2.TokenMetadata{TokenType: 3}}}
	_, err := NewRemoteVerifyStrategy(stub, testRecipientConfig()).Verify(context.Background(), token, nil)
	require.ErrorIs(t, err, iamerrors.ErrTokenInvalid)
}

func TestMalformedTypeCannotBecomeLegacyAccess(t *testing.T) {
	key, manager := newRS256Fixture(t)
	for _, kind := range []interface{}{3, true, []string{"access"}} {
		token := signRS256Token(t, key, map[string]interface{}{jwt.IssuerKey: "https://iam.fangcunmount.cn", jwt.AudienceKey: []string{"qs-api"}, jwt.SubjectKey: "1", jwt.ExpirationKey: time.Now().Add(time.Minute), "token_type": kind})
		_, err := NewLocalVerifyStrategy(manager, WithLocalConfig(testRecipientConfig())).Verify(context.Background(), token, nil)
		require.ErrorIs(t, err, iamerrors.ErrTokenInvalid)
	}
}
