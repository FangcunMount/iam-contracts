package verifier

import (
	"context"
	authnv2 "github.com/FangcunMount/iam/v4/api/grpc/iam/authn/v2"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestStandaloneStrategiesRequireEffectiveAudience(t *testing.T) {
	key, manager := newRS256Fixture(t)
	raw := signRS256Token(t, key, map[string]interface{}{jwt.IssuerKey: "https://iam.fangcunmount.cn", jwt.AudienceKey: []string{"qs-api"}, jwt.SubjectKey: "1", jwt.ExpirationKey: time.Now().Add(time.Minute)})
	for _, aud := range [][]string{{}, {" "}, {"qs-api", ""}} {
		opts := &VerifyOptions{ExpectedIssuer: "https://iam.fangcunmount.cn", ExpectedAudience: aud}
		_, err := NewLocalVerifyStrategy(manager).Verify(context.Background(), raw, opts)
		require.Error(t, err)
		stub := &verifyTokenClientStub{}
		_, err = NewRemoteVerifyStrategy(stub, nil).Verify(context.Background(), raw, opts)
		require.Error(t, err)
		require.Nil(t, stub.verifyReq)
	}
	_, err := NewLocalVerifyStrategy(manager).Verify(context.Background(), raw, nil)
	require.Error(t, err)
}

func TestLocalAndRemoteAudienceConstraintsUseAnyMatch(t *testing.T) {
	key, manager := newRS256Fixture(t)
	raw := signRS256Token(t, key, map[string]interface{}{jwt.IssuerKey: "https://iam.fangcunmount.cn", jwt.AudienceKey: []string{"qs-api"}, jwt.SubjectKey: "1", jwt.ExpirationKey: time.Now().Add(time.Minute)})
	for _, aud := range [][]string{{"collection-api", "qs-api"}, {"qs-api", "collection-api"}, {" qs-api ", "qs-api"}} {
		opts := &VerifyOptions{ExpectedAudience: aud}
		result, err := NewLocalVerifyStrategy(manager, WithLocalConfig(testRecipientConfig())).Verify(context.Background(), raw, opts)
		require.NoError(t, err)
		require.True(t, result.Valid)
		stub := &verifyTokenClientStub{verifyResp: &authnv2.VerifyTokenResponse{Valid: true, Claims: &authnv2.TokenClaims{TokenType: authnv2.TokenType_TOKEN_TYPE_ACCESS, Audience: []string{"qs-api"}}}}
		result, err = NewRemoteVerifyStrategy(stub, testRecipientConfig()).Verify(context.Background(), raw, opts)
		require.NoError(t, err)
		require.True(t, result.Valid)
		require.NotEmpty(t, stub.verifyReq.ExpectedAudience)
	}
	stub := &verifyTokenClientStub{}
	_, err := NewRemoteVerifyStrategy(stub, testRecipientConfig()).Verify(context.Background(), raw, &VerifyOptions{ExpectedAudience: []string{"iam-api"}})
	require.Error(t, err)
	require.Nil(t, stub.verifyReq)
}
