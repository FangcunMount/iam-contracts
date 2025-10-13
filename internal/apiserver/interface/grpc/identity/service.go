package identity

import (
	"context"
	"math"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	childdomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child"
	childport "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/child/port"
	guardianshipdomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/guardianship"
	guardport "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/guardianship/port"
	userdomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	userport "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user/port"
	identityv1 "github.com/fangcun-mount/iam-contracts/internal/apiserver/interface/grpc/pb/iam/identity/v1"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// Service 聚合 identity 模块的 gRPC 服务
type Service struct {
	identityRead    identityReadServer
	guardianshipQry guardianshipQueryServer
}

// NewService 创建 identity gRPC 服务
func NewService(
	userQuery userport.UserQueryer,
	childQuery childport.ChildQueryer,
	guardQuery guardport.GuardianshipQueryer,
) *Service {
	return &Service{
		identityRead: identityReadServer{
			userQuery:  userQuery,
			childQuery: childQuery,
		},
		guardianshipQry: guardianshipQueryServer{
			childQuery: childQuery,
			guardQuery: guardQuery,
		},
	}
}

// RegisterService 注册 gRPC 服务
func (s *Service) RegisterService(server *grpc.Server) {
	identityv1.RegisterIdentityReadServer(server, &s.identityRead)
	identityv1.RegisterGuardianshipQueryServer(server, &s.guardianshipQry)
}

type identityReadServer struct {
	identityv1.UnimplementedIdentityReadServer
	userQuery  userport.UserQueryer
	childQuery childport.ChildQueryer
}

type guardianshipQueryServer struct {
	identityv1.UnimplementedGuardianshipQueryServer
	childQuery childport.ChildQueryer
	guardQuery guardport.GuardianshipQueryer
}

// GetUser 查询用户
func (s *identityReadServer) GetUser(ctx context.Context, req *identityv1.GetUserReq) (*identityv1.GetUserResp, error) {
	if req == nil || strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID, err := parseUserID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	u, err := s.userQuery.FindByID(ctx, userID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &identityv1.GetUserResp{User: toProtoUser(u)}, nil
}

// GetChild 查询儿童档案
func (s *identityReadServer) GetChild(ctx context.Context, req *identityv1.GetChildReq) (*identityv1.GetChildResp, error) {
	if req == nil || strings.TrimSpace(req.GetChildId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "child_id is required")
	}

	childID, err := parseChildID(req.GetChildId())
	if err != nil {
		return nil, err
	}

	child, err := s.childQuery.FindByID(ctx, childID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &identityv1.GetChildResp{Child: toProtoChild(child)}, nil
}

// IsGuardian 判定是否为监护人
func (s *guardianshipQueryServer) IsGuardian(ctx context.Context, req *identityv1.IsGuardianReq) (*identityv1.IsGuardianResp, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	userID, err := parseUserID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	childID, err := parseChildID(req.GetChildId())
	if err != nil {
		return nil, err
	}

	guardianship, err := s.guardQuery.FindByUserIDAndChildID(ctx, userID, childID)
	if err != nil {
		if coder := errors.ParseCoder(err); coder != nil && coder.HTTPStatus() == 404 {
			return &identityv1.IsGuardianResp{IsGuardian: false}, nil
		}
		return nil, toGRPCError(err)
	}

	return &identityv1.IsGuardianResp{IsGuardian: guardianship != nil && guardianship.IsActive()}, nil
}

// ListChildren 列出监护的儿童
func (s *guardianshipQueryServer) ListChildren(ctx context.Context, req *identityv1.ListChildrenReq) (*identityv1.ListChildrenResp, error) {
	if req == nil || strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID, err := parseUserID(req.GetUserId())
	if err != nil {
		return nil, err
	}

	limit := int(req.GetLimit())
	offset := int(req.GetOffset())
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		return nil, status.Error(codes.InvalidArgument, "offset must be >= 0")
	}

	guardianships, err := s.guardQuery.FindListByUserID(ctx, userID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	total := len(guardianships)
	if offset >= total {
		return &identityv1.ListChildrenResp{
			Total: int32(total),
			Items: []*identityv1.Child{},
		}, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	items := make([]*identityv1.Child, 0, end-offset)
	for _, g := range guardianships[offset:end] {
		if g == nil {
			continue
		}

		child, err := s.childQuery.FindByID(ctx, g.Child)
		if err != nil {
			return nil, toGRPCError(err)
		}
		items = append(items, toProtoChild(child))
	}

	return &identityv1.ListChildrenResp{
		Total: int32(total),
		Items: items,
	}, nil
}

func parseUserID(raw string) (userdomain.UserID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return userdomain.UserID{}, status.Error(codes.InvalidArgument, "user_id cannot be empty")
	}

	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return userdomain.UserID{}, status.Errorf(codes.InvalidArgument, "invalid user_id: %s", raw)
	}

	return userdomain.NewUserID(id), nil
}

func parseChildID(raw string) (childdomain.ChildID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return childdomain.ChildID{}, status.Error(codes.InvalidArgument, "child_id cannot be empty")
	}

	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return childdomain.ChildID{}, status.Errorf(codes.InvalidArgument, "invalid child_id: %s", raw)
	}

	return childdomain.NewChildID(id), nil
}

func toProtoUser(u *userdomain.User) *identityv1.User {
	if u == nil {
		return nil
	}

	user := &identityv1.User{
		Id:       u.ID.String(),
		Status:   u.Status.String(),
		Nickname: u.Name,
		Avatar:   "",
	}

	return user
}

func toProtoChild(c *childdomain.Child) *identityv1.Child {
	if c == nil {
		return nil
	}

	var (
		height int32
		weight string
	)

	if c.Height.Tenths() > 0 {
		height = int32(math.Round(c.Height.Float()))
	}
	if c.Weight.Tenths() > 0 {
		weight = c.Weight.String()
	}

	child := &identityv1.Child{
		Id:        c.ID.String(),
		LegalName: c.Name,
		Gender:    int32(c.Gender.Value()),
		Dob:       c.Birthday.String(),
		IdType:    "",
		HeightCm:  height,
		WeightKg:  weight,
	}

	return child
}

func relationToString(rel guardianshipdomain.Relation) string {
	switch rel {
	case guardianshipdomain.RelSelf:
		return "self"
	case guardianshipdomain.RelParent:
		return "parent"
	default:
		return "guardian"
	}
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
		case 409:
			return status.Error(codes.AlreadyExists, coder.String())
		default:
			return status.Error(codes.Internal, coder.String())
		}
	}

	return status.Error(codes.Internal, err.Error())
}
