package authn

import (
	"context"

	authnv3 "github.com/FangcunMount/iam/v5/api/grpc/iam/authn/v3"
	linkingApp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authn/linking"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *loginIdentityServiceServer) ListLoginIdentities(ctx context.Context, req *authnv3.ListLoginIdentitiesRequest) (*authnv3.ListLoginIdentitiesResponse, error) {
	if s.linking == nil {
		return nil, status.Error(codes.Unimplemented, "login identity service not configured")
	}
	userID, _, _, err := parseAuthenticatedUserContext(req.GetActor())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	identities, err := s.linking.List(ctx, userID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	out := make([]*authnv3.LoginIdentity, 0, len(identities))
	for _, identity := range identities {
		out = append(out, toProtoLoginIdentityView(identity))
	}
	return &authnv3.ListLoginIdentitiesResponse{Items: out}, nil
}

func (s *loginIdentityServiceServer) SendPhoneLinkChallenge(ctx context.Context, req *authnv3.SendPhoneLinkChallengeRequest) (*authnv3.MessageResponse, error) {
	if s.linking == nil {
		return nil, status.Error(codes.Unimplemented, "login identity service not configured")
	}
	if _, _, _, err := parseAuthenticatedUserContext(req.GetActor()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if s.phoneLinkOTPSender == nil {
		return nil, status.Error(codes.Unimplemented, "phone link challenge is not configured")
	}
	if err := s.phoneLinkOTPSender.SendPhoneLinkOTP(ctx, req.GetPhone()); err != nil {
		return nil, toGRPCError(err)
	}
	return &authnv3.MessageResponse{Message: "verification code sent"}, nil
}

func (s *loginIdentityServiceServer) LinkPhone(ctx context.Context, req *authnv3.LinkPhoneRequest) (*authnv3.LinkLoginIdentityResponse, error) {
	if s.linking == nil {
		return nil, status.Error(codes.Unimplemented, "login identity service not configured")
	}
	userID, _, authenticatedAt, err := parseAuthenticatedUserContext(req.GetActor())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	result, err := s.linking.Link(ctx, linkingApp.LinkRequest{
		AuthenticatedAt: authenticatedAt,
		UserID:          userID,
		Input: linkingApp.LinkPhoneInput{
			Phone:   req.GetPhone(),
			OTPCode: req.GetOtpCode(),
		},
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoLinkResult(result), nil
}

func (s *loginIdentityServiceServer) LinkWechatMiniProgram(ctx context.Context, req *authnv3.LinkWechatMiniProgramRequest) (*authnv3.LinkLoginIdentityResponse, error) {
	if s.linking == nil {
		return nil, status.Error(codes.Unimplemented, "login identity service not configured")
	}
	userID, _, authenticatedAt, err := parseAuthenticatedUserContext(req.GetActor())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	result, err := s.linking.Link(ctx, linkingApp.LinkRequest{
		AuthenticatedAt: authenticatedAt,
		UserID:          userID,
		Input: linkingApp.LinkWechatMiniInput{
			AppID: req.GetAppId(),
			Code:  req.GetCode(),
		},
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoLinkResult(result), nil
}

func (s *loginIdentityServiceServer) LinkWecom(ctx context.Context, req *authnv3.LinkWecomRequest) (*authnv3.LinkLoginIdentityResponse, error) {
	if s.linking == nil {
		return nil, status.Error(codes.Unimplemented, "login identity service not configured")
	}
	userID, _, authenticatedAt, err := parseAuthenticatedUserContext(req.GetActor())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	result, err := s.linking.Link(ctx, linkingApp.LinkRequest{
		AuthenticatedAt: authenticatedAt,
		UserID:          userID,
		Input: linkingApp.LinkWecomInput{
			CorpID: req.GetCorpId(),
			Code:   req.GetCode(),
		},
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toProtoLinkResult(result), nil
}

func (s *loginIdentityServiceServer) UnlinkLoginIdentity(ctx context.Context, req *authnv3.UnlinkLoginIdentityRequest) (*authnv3.MessageResponse, error) {
	if s.linking == nil {
		return nil, status.Error(codes.Unimplemented, "login identity service not configured")
	}
	userID, currentID, authenticatedAt, err := parseAuthenticatedUserContext(req.GetActor())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	loginIdentityID, err := parseRequiredMetaID(req.GetLoginIdentityId(), "login_identity_id")
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.linking.Unlink(ctx, linkingApp.UnlinkCommand{
		UserID:                 userID,
		LoginIdentityID:        loginIdentityID,
		CurrentLoginIdentityID: currentID,
		AuthenticatedAt:        authenticatedAt,
	}); err != nil {
		return nil, toGRPCError(err)
	}
	return &authnv3.MessageResponse{Message: "login identity unlinked"}, nil
}
