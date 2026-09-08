package sdk

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	authnv2 "github.com/FangcunMount/iam/v4/api/grpc/iam/authn/v2"
	authclient "github.com/FangcunMount/iam/v4/pkg/sdk/auth/client"
	authjwks "github.com/FangcunMount/iam/v4/pkg/sdk/auth/jwks"
	authverifier "github.com/FangcunMount/iam/v4/pkg/sdk/auth/verifier"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type sdkAuthServiceClientStub struct {
	verifyReq  *authnv2.VerifyTokenRequest
	verifyResp *authnv2.VerifyTokenResponse
	verifyErr  error
}

func (s *sdkAuthServiceClientStub) VerifyToken(ctx context.Context, in *authnv2.VerifyTokenRequest, _ ...grpc.CallOption) (*authnv2.VerifyTokenResponse, error) {
	s.verifyReq = in
	return s.verifyResp, s.verifyErr
}

func (s *sdkAuthServiceClientStub) Login(context.Context, *authnv2.LoginRequest, ...grpc.CallOption) (*authnv2.LoginResponse, error) {
	return nil, nil
}

func (s *sdkAuthServiceClientStub) RefreshToken(context.Context, *authnv2.RefreshTokenRequest, ...grpc.CallOption) (*authnv2.RefreshTokenResponse, error) {
	return nil, nil
}

func (s *sdkAuthServiceClientStub) RevokeToken(context.Context, *authnv2.RevokeTokenRequest, ...grpc.CallOption) (*authnv2.RevokeTokenResponse, error) {
	return nil, nil
}

func (s *sdkAuthServiceClientStub) RevokeRefreshToken(context.Context, *authnv2.RevokeRefreshTokenRequest, ...grpc.CallOption) (*authnv2.RevokeRefreshTokenResponse, error) {
	return nil, nil
}

type sdkJWKSServiceClientStub struct {
	resp *authnv2.GetJWKSResponse
	err  error
}

func (s *sdkJWKSServiceClientStub) GetJWKS(context.Context, *authnv2.GetJWKSRequest, ...grpc.CallOption) (*authnv2.GetJWKSResponse, error) {
	return s.resp, s.err
}

func TestClientAuthUsesTypedAuthClient(t *testing.T) {
	t.Parallel()

	authStub := &sdkAuthServiceClientStub{
		verifyResp: &authnv2.VerifyTokenResponse{
			Valid: true,
			Claims: &authnv2.TokenClaims{
				TokenId:   "jti-1",
				Subject:   "user:1",
				SessionId: "sid-1",
				UserId:    "1",
				Issuer:    "https://iam.example.com",
				Audience:  []string{"qs-api"},
				IssuedAt:  timestamppb.New(time.Now()),
				ExpiresAt: timestamppb.New(time.Now().Add(time.Minute)),
			},
		},
	}

	client := &Client{
		authClient: authclient.NewClient(authStub, &sdkJWKSServiceClientStub{}),
	}

	resp, err := client.Auth().VerifyToken(context.Background(), &authnv2.VerifyTokenRequest{
		AccessToken:      "jwt-token",
		ExpectedAudience: []string{"qs-api"},
	})
	require.NoError(t, err)
	require.True(t, resp.GetValid())
	require.NotNil(t, authStub.verifyReq)
}

func TestAuthSubpackagesComposeWithSDKClient(t *testing.T) {
	t.Parallel()

	authStub := &sdkAuthServiceClientStub{
		verifyResp: &authnv2.VerifyTokenResponse{
			Valid: true,
			Claims: &authnv2.TokenClaims{
				TokenId:   "jti-1",
				Subject:   "user:1",
				SessionId: "sid-1",
				UserId:    "1",
				Issuer:    "https://iam.example.com",
				Audience:  []string{"qs-api"},
				IssuedAt:  timestamppb.New(time.Now()),
				ExpiresAt: timestamppb.New(time.Now().Add(time.Minute)),
			},
		},
	}

	client := &Client{
		authClient: authclient.NewClient(authStub, &sdkJWKSServiceClientStub{
			resp: &authnv2.GetJWKSResponse{Jwks: []byte(`{"keys":[]}`)},
		}),
	}

	jwksManager, err := authjwks.NewJWKSManager(&JWKSConfig{
		URL:            "https://iam.example.com/.well-known/jwks.json",
		RequestTimeout: time.Second,
	}, authjwks.WithAuthClient(client.Auth()), authjwks.WithSeedData([]byte(`{"keys":[]}`)))
	require.NoError(t, err)
	defer jwksManager.Stop()

	verifier, err := authverifier.NewTokenVerifier(&TokenVerifyConfig{
		AllowedIssuer:   "https://iam.example.com",
		AllowedAudience: []string{"qs-api"},
	}, jwksManager, client.Auth())
	require.NoError(t, err)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	jwtToken := jwt.New()
	require.NoError(t, jwtToken.Set(jwt.SubjectKey, "user:1"))
	require.NoError(t, jwtToken.Set(jwt.IssuerKey, "https://iam.example.com"))
	require.NoError(t, jwtToken.Set(jwt.AudienceKey, []string{"qs-api"}))
	require.NoError(t, jwtToken.Set(jwt.ExpirationKey, time.Now().Add(time.Minute)))
	signedToken, err := jwt.Sign(jwtToken, jwt.WithKey(jwa.RS256, privateKey))
	require.NoError(t, err)

	result, err := verifier.Verify(context.Background(), string(signedToken), nil)
	require.NoError(t, err)
	require.Equal(t, "sid-1", result.Claims.SessionID)

}
