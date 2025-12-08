package grpc

import (
	"context"
	"strings"
	"time"

	"github.com/FangcunMount/component-base/pkg/errors"
	authnv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/authn/v1"
	jwksApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/jwks"
	tokenApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/token"
	tokenDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/token"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service 聚合 authn 模块的 gRPC 服务
type Service struct {
	auth authServiceServer
	jwks jwksServiceServer
}

// NewService 创建 authn gRPC 服务
func NewService(
	tokenSvc tokenApp.TokenApplicationService,
	keyPublish *jwksApp.KeyPublishAppService,
) *Service {
	return &Service{
		auth: authServiceServer{
			tokenSvc: tokenSvc,
		},
		jwks: jwksServiceServer{
			keyPublish: keyPublish,
		},
	}
}

// Register 注册 gRPC 服务
func (s *Service) Register(server *grpc.Server) {
	if s == nil || server == nil {
		return
	}
	if s.auth.tokenSvc != nil {
		authnv1.RegisterAuthServiceServer(server, &s.auth)
	}
	if s.jwks.keyPublish != nil {
		authnv1.RegisterJWKSServiceServer(server, &s.jwks)
	}
}

type authServiceServer struct {
	authnv1.UnimplementedAuthServiceServer
	tokenSvc tokenApp.TokenApplicationService
}

type jwksServiceServer struct {
	authnv1.UnimplementedJWKSServiceServer
	keyPublish *jwksApp.KeyPublishAppService
}

func (s *authServiceServer) VerifyToken(ctx context.Context, req *authnv1.VerifyTokenRequest) (*authnv1.VerifyTokenResponse, error) {
	if s.tokenSvc == nil {
		return nil, status.Error(codes.Unimplemented, "token service not configured")
	}
	if req == nil || strings.TrimSpace(req.GetAccessToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "access_token is required")
	}

	result, err := s.tokenSvc.VerifyToken(ctx, req.GetAccessToken())
	if err != nil {
		return nil, toGRPCError(err)
	}

	resp := &authnv1.VerifyTokenResponse{}
	if result != nil {
		resp.Valid = result.Valid
	}
	if resp.Valid {
		resp.Status = authnv1.TokenStatus_TOKEN_STATUS_VALID
		resp.Claims = toProtoTokenClaims(result.Claims)
		if req.GetIncludeMetadata() {
			resp.Metadata = buildTokenMetadata(result.Claims)
		}
	} else {
		resp.Status = authnv1.TokenStatus_TOKEN_STATUS_REVOKED
		resp.FailureReason = "token invalid or expired"
	}
	return resp, nil
}

func (s *authServiceServer) RefreshToken(ctx context.Context, req *authnv1.RefreshTokenRequest) (*authnv1.RefreshTokenResponse, error) {
	if s.tokenSvc == nil {
		return nil, status.Error(codes.Unimplemented, "token service not configured")
	}
	if req == nil || strings.TrimSpace(req.GetRefreshToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	result, err := s.tokenSvc.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authnv1.RefreshTokenResponse{
		TokenPair: toProtoTokenPair(result.TokenPair),
	}, nil
}

func (s *authServiceServer) RevokeToken(ctx context.Context, req *authnv1.RevokeTokenRequest) (*authnv1.RevokeTokenResponse, error) {
	if s.tokenSvc == nil {
		return nil, status.Error(codes.Unimplemented, "token service not configured")
	}
	if req == nil || strings.TrimSpace(req.GetAccessToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "access_token is required")
	}
	if err := s.tokenSvc.RevokeToken(ctx, req.GetAccessToken()); err != nil {
		return nil, toGRPCError(err)
	}
	return &authnv1.RevokeTokenResponse{}, nil
}

func (s *authServiceServer) RevokeRefreshToken(ctx context.Context, req *authnv1.RevokeRefreshTokenRequest) (*authnv1.RevokeRefreshTokenResponse, error) {
	if s.tokenSvc == nil {
		return nil, status.Error(codes.Unimplemented, "token service not configured")
	}
	if req == nil || strings.TrimSpace(req.GetRefreshToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}
	if err := s.tokenSvc.RevokeRefreshToken(ctx, req.GetRefreshToken()); err != nil {
		return nil, toGRPCError(err)
	}
	return &authnv1.RevokeRefreshTokenResponse{}, nil
}

func (s *authServiceServer) IssueServiceToken(context.Context, *authnv1.IssueServiceTokenRequest) (*authnv1.IssueServiceTokenResponse, error) {
	return nil, status.Error(codes.Unimplemented, "issue service token not supported")
}

func (s *jwksServiceServer) GetJWKS(ctx context.Context, req *authnv1.GetJWKSRequest) (*authnv1.GetJWKSResponse, error) {
	if s.keyPublish == nil {
		return nil, status.Error(codes.Unimplemented, "jwks service not configured")
	}
	_ = req // reserved for future cache validation
	result, err := s.keyPublish.BuildJWKS(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &authnv1.GetJWKSResponse{
		Jwks:         result.JWKS,
		Etag:         result.ETag,
		LastModified: timestamppb.New(result.LastModified),
	}, nil
}

func toProtoTokenPair(pair *tokenDomain.TokenPair) *authnv1.TokenPair {
	if pair == nil || pair.AccessToken == nil {
		return nil
	}
	resp := &authnv1.TokenPair{
		TokenType:    "Bearer",
		AccessToken:  pair.AccessToken.Value,
		RefreshToken: "",
		ExpiresIn:    durationpb.New(durationUntil(pair.AccessToken.ExpiresAt)),
	}
	if pair.RefreshToken != nil {
		resp.RefreshToken = pair.RefreshToken.Value
	}
	return resp
}

func toProtoTokenClaims(claims *tokenDomain.TokenClaims) *authnv1.TokenClaims {
	if claims == nil {
		return nil
	}
	return &authnv1.TokenClaims{
		TokenId:   claims.TokenID,
		UserId:    claims.UserID.String(),
		AccountId: claims.AccountID.String(),
		IssuedAt:  timestamppb.New(claims.IssuedAt),
		ExpiresAt: timestamppb.New(claims.ExpiresAt),
	}
}

func buildTokenMetadata(claims *tokenDomain.TokenClaims) *authnv1.TokenMetadata {
	if claims == nil {
		return nil
	}
	return &authnv1.TokenMetadata{
		TokenType: authnv1.TokenType_TOKEN_TYPE_ACCESS,
		Status:    authnv1.TokenStatus_TOKEN_STATUS_VALID,
		IssuedAt:  timestamppb.New(claims.IssuedAt),
		ExpiresAt: timestamppb.New(claims.ExpiresAt),
	}
}

func durationUntil(t time.Time) time.Duration {
	d := time.Until(t)
	if d < 0 {
		return 0
	}
	return d
}

func toGRPCError(err error) error {
	if err == nil {
		return nil
	}
	if coder := errors.ParseCoder(err); coder != nil {
		switch coder.HTTPStatus() {
		case 400:
			return status.Error(codes.InvalidArgument, coder.String())
		case 401:
			return status.Error(codes.Unauthenticated, coder.String())
		case 403:
			return status.Error(codes.PermissionDenied, coder.String())
		case 404:
			return status.Error(codes.NotFound, coder.String())
		}
		return status.Error(codes.Internal, coder.String())
	}
	return status.Error(codes.Internal, err.Error())
}
