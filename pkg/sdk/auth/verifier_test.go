package auth

import (
	"context"
	"testing"
	"time"

	authnv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/authn/v1"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/config"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type authServiceClientStub struct {
	verifyReq  *authnv1.VerifyTokenRequest
	verifyResp *authnv1.VerifyTokenResponse
	verifyErr  error
}

func (s *authServiceClientStub) VerifyToken(ctx context.Context, in *authnv1.VerifyTokenRequest, opts ...grpc.CallOption) (*authnv1.VerifyTokenResponse, error) {
	s.verifyReq = in
	return s.verifyResp, s.verifyErr
}

func (s *authServiceClientStub) RefreshToken(context.Context, *authnv1.RefreshTokenRequest, ...grpc.CallOption) (*authnv1.RefreshTokenResponse, error) {
	return nil, nil
}

func (s *authServiceClientStub) RevokeToken(context.Context, *authnv1.RevokeTokenRequest, ...grpc.CallOption) (*authnv1.RevokeTokenResponse, error) {
	return nil, nil
}

func (s *authServiceClientStub) RevokeRefreshToken(context.Context, *authnv1.RevokeRefreshTokenRequest, ...grpc.CallOption) (*authnv1.RevokeRefreshTokenResponse, error) {
	return nil, nil
}

func (s *authServiceClientStub) IssueServiceToken(context.Context, *authnv1.IssueServiceTokenRequest, ...grpc.CallOption) (*authnv1.IssueServiceTokenResponse, error) {
	return nil, nil
}

func TestRemoteVerifyStrategyPassesConfiguredIssuerAndAudience(t *testing.T) {
	stub := &authServiceClientStub{
		verifyResp: &authnv1.VerifyTokenResponse{
			Valid: true,
			Claims: &authnv1.TokenClaims{
				Subject:   "user:1",
				UserId:    "1",
				AccountId: "2",
				TenantId:  "3",
				Issuer:    "https://iam.fangcunmount.cn",
				Audience:  []string{"qs-api", "collection-api"},
				IssuedAt:  timestamppb.New(time.Now()),
				ExpiresAt: timestamppb.New(time.Now().Add(time.Minute)),
			},
		},
	}

	strategy := NewRemoteVerifyStrategy(&Client{authService: stub}, &config.TokenVerifyConfig{
		AllowedIssuer:   "https://iam.fangcunmount.cn",
		AllowedAudience: []string{"qs-api"},
	})

	_, err := strategy.Verify(context.Background(), "jwt-token", nil)
	require.NoError(t, err)
	require.NotNil(t, stub.verifyReq)
	require.Equal(t, "https://iam.fangcunmount.cn", stub.verifyReq.ExpectedIssuer)
	require.Equal(t, []string{"qs-api"}, stub.verifyReq.ExpectedAudience)
}

func TestRemoteVerifyStrategyOptionsOverrideConfig(t *testing.T) {
	stub := &authServiceClientStub{
		verifyResp: &authnv1.VerifyTokenResponse{
			Valid: true,
			Claims: &authnv1.TokenClaims{
				Subject:   "user:1",
				UserId:    "1",
				AccountId: "2",
				TenantId:  "3",
				Issuer:    "https://issuer.override",
				Audience:  []string{"collection-api"},
				IssuedAt:  timestamppb.New(time.Now()),
				ExpiresAt: timestamppb.New(time.Now().Add(time.Minute)),
			},
		},
	}

	strategy := NewRemoteVerifyStrategy(&Client{authService: stub}, &config.TokenVerifyConfig{
		AllowedIssuer:   "https://iam.fangcunmount.cn",
		AllowedAudience: []string{"qs-api"},
	})

	_, err := strategy.Verify(context.Background(), "jwt-token", &VerifyOptions{
		ExpectedIssuer:   "https://issuer.override",
		ExpectedAudience: []string{"collection-api"},
	})
	require.NoError(t, err)
	require.NotNil(t, stub.verifyReq)
	require.Equal(t, "https://issuer.override", stub.verifyReq.ExpectedIssuer)
	require.Equal(t, []string{"collection-api"}, stub.verifyReq.ExpectedAudience)
}
