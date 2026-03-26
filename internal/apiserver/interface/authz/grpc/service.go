package grpc

import (
	"context"

	authzv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/authz/v1"
	policyDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/policy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service 聚合 authz gRPC（PDP）。
type Service struct {
	srv authorizationServer
}

// NewService 创建 authz gRPC 服务。
func NewService(casbin policyDomain.CasbinAdapter) *Service {
	return &Service{
		srv: authorizationServer{casbin: casbin},
	}
}

// Register 注册到 gRPC Server。
func (s *Service) Register(server *grpc.Server) {
	if s == nil || server == nil {
		return
	}
	authzv1.RegisterAuthorizationServiceServer(server, &s.srv)
}

type authorizationServer struct {
	authzv1.UnimplementedAuthorizationServiceServer
	casbin policyDomain.CasbinAdapter
}

func (s *authorizationServer) Check(ctx context.Context, req *authzv1.CheckRequest) (*authzv1.CheckResponse, error) {
	if s.casbin == nil {
		return nil, status.Error(codes.Unavailable, "authorization engine not available")
	}
	if req == nil || req.Subject == "" || req.Domain == "" || req.Object == "" || req.Action == "" {
		return nil, status.Error(codes.InvalidArgument, "subject, domain, object, action are required")
	}
	ok, err := s.casbin.Enforce(ctx, req.Subject, req.Domain, req.Object, req.Action)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "enforce: %v", err)
	}
	return &authzv1.CheckResponse{Allowed: ok}, nil
}
