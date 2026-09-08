package authn

import (
	"context"
	"strings"

	authnv2 "github.com/FangcunMount/iam/v4/api/grpc/iam/authn/v2"
	tokenApp "github.com/FangcunMount/iam/v4/internal/apiserver/application/authn/token"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *authServiceServer) VerifyToken(ctx context.Context, req *authnv2.VerifyTokenRequest) (*authnv2.VerifyTokenResponse, error) {
	if s.tokenVerifier == nil {
		return nil, status.Error(codes.Unimplemented, "token verifier not configured")
	}
	if req == nil || strings.TrimSpace(req.GetAccessToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "access_token is required")
	}

	result, err := s.tokenVerifier.VerifyToken(ctx, tokenApp.VerifyTokenRequest{
		AccessToken:        req.GetAccessToken(),
		ExpectedIssuer:     strings.TrimSpace(req.GetExpectedIssuer()),
		ExpectedAudience:   cloneAudience(req.GetExpectedAudience()),
		AcceptedTokenTypes: acceptedDomainTokenTypes(req.GetAcceptedTokenTypes()),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	resp := &authnv2.VerifyTokenResponse{}
	if result != nil {
		resp.Valid = result.Valid
	}
	if resp.Valid {
		resp.Status = authnv2.TokenStatus_TOKEN_STATUS_VALID
		resp.Claims = toProtoTokenClaims(result.Claims)
		if req.GetIncludeMetadata() {
			resp.Metadata = buildTokenMetadata(result.Claims)
		}
	} else {
		resp.Status = authnv2.TokenStatus_TOKEN_STATUS_REVOKED
		resp.FailureReason = "token invalid or expired"
	}
	return resp, nil
}

func acceptedDomainTokenTypes(values []authnv2.TokenType) []tokenApp.TokenType {
	if len(values) == 0 {
		return []tokenApp.TokenType{tokenApp.TokenTypeAccess}
	}
	out := make([]tokenApp.TokenType, 0, len(values))
	for _, value := range values {
		switch value {
		case authnv2.TokenType_TOKEN_TYPE_ACCESS:
			out = append(out, tokenApp.TokenTypeAccess)
		case authnv2.TokenType_TOKEN_TYPE_REFRESH:
			out = append(out, tokenApp.TokenTypeRefresh)
		default:
			out = append(out, tokenApp.TokenType("invalid"))
		}
	}
	return out
}

func (s *authServiceServer) RefreshToken(ctx context.Context, req *authnv2.RefreshTokenRequest) (*authnv2.RefreshTokenResponse, error) {
	if s.sessionSvc == nil {
		return nil, status.Error(codes.Unimplemented, "session service not configured")
	}
	if req == nil || strings.TrimSpace(req.GetRefreshToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	result, err := s.sessionSvc.RenewSession(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &authnv2.RefreshTokenResponse{
		TokenPair: toProtoTokenPair(result.TokenPair),
	}, nil
}

func (s *authServiceServer) RevokeToken(ctx context.Context, req *authnv2.RevokeTokenRequest) (*authnv2.RevokeTokenResponse, error) {
	if s.tokenRevoker == nil {
		return nil, status.Error(codes.Unimplemented, "token revoker not configured")
	}
	if req == nil || strings.TrimSpace(req.GetAccessToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "access_token is required")
	}
	if err := s.tokenRevoker.RevokeAccessToken(ctx, req.GetAccessToken()); err != nil {
		return nil, toGRPCError(err)
	}
	return &authnv2.RevokeTokenResponse{}, nil
}

func (s *authServiceServer) RevokeRefreshToken(ctx context.Context, req *authnv2.RevokeRefreshTokenRequest) (*authnv2.RevokeRefreshTokenResponse, error) {
	if s.tokenRevoker == nil {
		return nil, status.Error(codes.Unimplemented, "token revoker not configured")
	}
	if req == nil || strings.TrimSpace(req.GetRefreshToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}
	if err := s.tokenRevoker.RevokeRefreshToken(ctx, req.GetRefreshToken()); err != nil {
		return nil, toGRPCError(err)
	}
	return &authnv2.RevokeRefreshTokenResponse{}, nil
}
