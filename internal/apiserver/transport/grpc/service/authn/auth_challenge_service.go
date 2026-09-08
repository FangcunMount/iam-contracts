package authn

import (
	"context"

	authnv3 "github.com/FangcunMount/iam/v5/api/grpc/iam/authn/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *authChallengeServiceServer) SendLoginPhoneOTP(ctx context.Context, req *authnv3.SendLoginPhoneOTPRequest) (*authnv3.MessageResponse, error) {
	if s.loginPhoneOTPSender == nil {
		return nil, status.Error(codes.Unimplemented, "challenge service not configured")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if err := s.loginPhoneOTPSender.SendLoginPhoneOTP(ctx, req.GetPhone()); err != nil {
		return nil, toGRPCError(err)
	}
	return &authnv3.MessageResponse{Message: "verification code sent"}, nil
}
