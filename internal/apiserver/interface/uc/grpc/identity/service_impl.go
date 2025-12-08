package identity

import (
	"context"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/identity/v1"
	guardianshipApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/guardianship"
	userApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/user"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ============= IdentityRead 服务实现 =============

// GetUser 查询用户
func (s *identityReadServer) GetUser(ctx context.Context, req *identityv1.GetUserRequest) (*identityv1.GetUserResponse, error) {
	if req == nil || strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	result, err := s.userQuerySvc.GetByID(ctx, req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &identityv1.GetUserResponse{User: userResultToProto(result)}, nil
}

// BatchGetUsers 批量查询用户
func (s *identityReadServer) BatchGetUsers(ctx context.Context, req *identityv1.BatchGetUsersRequest) (*identityv1.BatchGetUsersResponse, error) {
	if req == nil || len(req.GetUserIds()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_ids is required")
	}

	resp := &identityv1.BatchGetUsersResponse{
		Users:       make([]*identityv1.User, 0, len(req.GetUserIds())),
		NotFoundIds: make([]string, 0),
	}

	for _, userID := range req.GetUserIds() {
		result, err := s.userQuerySvc.GetByID(ctx, userID)
		if err != nil {
			// 如果是未找到错误，添加到 not_found 列表
			resp.NotFoundIds = append(resp.NotFoundIds, userID)
			continue
		}
		resp.Users = append(resp.Users, userResultToProto(result))
	}

	return resp, nil
}

// SearchUsers 搜索用户
func (s *identityReadServer) SearchUsers(ctx context.Context, req *identityv1.SearchUsersRequest) (*identityv1.SearchUsersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	// 目前只支持通过手机号精确查询
	if len(req.GetPhones()) > 0 {
		return s.searchUsersByPhones(ctx, req)
	}

	// 其他搜索条件暂不支持
	return &identityv1.SearchUsersResponse{
		Total: 0,
		Page:  req.GetPage(),
		Users: []*identityv1.User{},
	}, nil
}

// searchUsersByPhones 通过手机号列表搜索用户
func (s *identityReadServer) searchUsersByPhones(ctx context.Context, req *identityv1.SearchUsersRequest) (*identityv1.SearchUsersResponse, error) {
	users := make([]*identityv1.User, 0)

	for _, phone := range req.GetPhones() {
		result, err := s.userQuerySvc.GetByPhone(ctx, phone)
		if err != nil {
			// 忽略未找到的错误
			continue
		}
		users = append(users, userResultToProto(result))
	}

	return &identityv1.SearchUsersResponse{
		Total: int32(len(users)),
		Page:  req.GetPage(),
		Users: users,
	}, nil
}

// GetChild 查询儿童档案
func (s *identityReadServer) GetChild(ctx context.Context, req *identityv1.GetChildRequest) (*identityv1.GetChildResponse, error) {
	if req == nil || strings.TrimSpace(req.GetChildId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "child_id is required")
	}

	result, err := s.childQuerySvc.GetByID(ctx, req.GetChildId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &identityv1.GetChildResponse{Child: childResultToProto(result)}, nil
}

// BatchGetChildren 批量查询儿童档案
func (s *identityReadServer) BatchGetChildren(ctx context.Context, req *identityv1.BatchGetChildrenRequest) (*identityv1.BatchGetChildrenResponse, error) {
	if req == nil || len(req.GetChildIds()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "child_ids is required")
	}

	resp := &identityv1.BatchGetChildrenResponse{
		Children:    make([]*identityv1.Child, 0, len(req.GetChildIds())),
		NotFoundIds: make([]string, 0),
	}

	for _, childID := range req.GetChildIds() {
		result, err := s.childQuerySvc.GetByID(ctx, childID)
		if err != nil {
			// 如果是未找到错误，添加到 not_found 列表
			resp.NotFoundIds = append(resp.NotFoundIds, childID)
			continue
		}
		resp.Children = append(resp.Children, childResultToProto(result))
	}

	return resp, nil
}

// ============= GuardianshipQuery 服务实现 =============

// IsGuardian 判定是否为监护人
func (s *guardianshipQueryServer) IsGuardian(ctx context.Context, req *identityv1.IsGuardianRequest) (*identityv1.IsGuardianResponse, error) {
	if req == nil || strings.TrimSpace(req.GetUserId()) == "" || strings.TrimSpace(req.GetChildId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and child_id are required")
	}

	isGuardian, err := s.guardianshipQuerySvc.IsGuardian(ctx, req.GetUserId(), req.GetChildId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	resp := &identityv1.IsGuardianResponse{IsGuardian: isGuardian}

	// 如果是监护人，返回监护关系详情
	if isGuardian {
		guardianship, err := s.guardianshipQuerySvc.GetByUserIDAndChildID(ctx, req.GetUserId(), req.GetChildId())
		if err == nil && guardianship != nil {
			resp.Guardianship = guardianshipResultToProto(guardianship)
		}
	}

	return resp, nil
}

// ListChildren 列出监护的儿童
func (s *guardianshipQueryServer) ListChildren(ctx context.Context, req *identityv1.ListChildrenRequest) (*identityv1.ListChildrenResponse, error) {
	if req == nil || strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	guardianships, err := s.guardianshipQuerySvc.ListChildrenByUserID(ctx, req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	// 应用分页
	limit := int(req.GetPage().GetLimit())
	offset := int(req.GetPage().GetOffset())
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	total := len(guardianships)
	items := make([]*identityv1.ChildEdge, 0)

	if offset < total {
		end := offset + limit
		if end > total {
			end = total
		}
		for _, g := range guardianships[offset:end] {
			items = append(items, &identityv1.ChildEdge{
				Child:        childResultToProtoFromGuardianship(g),
				Guardianship: guardianshipResultToProto(g),
			})
		}
	}

	return &identityv1.ListChildrenResponse{
		Total: int32(total),
		Page:  req.GetPage(),
		Items: items,
	}, nil
}

// ListGuardians 列出儿童的所有监护人
func (s *guardianshipQueryServer) ListGuardians(ctx context.Context, req *identityv1.ListGuardiansRequest) (*identityv1.ListGuardiansResponse, error) {
	if req == nil || strings.TrimSpace(req.GetChildId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "child_id is required")
	}

	guardianships, err := s.guardianshipQuerySvc.ListGuardiansByChildID(ctx, req.GetChildId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	items := make([]*identityv1.GuardianshipEdge, 0, len(guardianships))
	for _, g := range guardianships {
		items = append(items, &identityv1.GuardianshipEdge{
			Guardianship: guardianshipResultToProto(g),
			Guardian:     nil, // 需要额外查询用户信息，暂不实现
		})
	}

	return &identityv1.ListGuardiansResponse{
		Total: int32(len(items)),
		Items: items,
	}, nil
}

// ============= GuardianshipCommand 服务实现 =============

// AddGuardian 添加监护人
func (s *guardianshipCommandServer) AddGuardian(ctx context.Context, req *identityv1.AddGuardianRequest) (*identityv1.AddGuardianResponse, error) {
	if req == nil || strings.TrimSpace(req.GetUserId()) == "" || strings.TrimSpace(req.GetChildId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and child_id are required")
	}

	dto := guardianshipApp.AddGuardianDTO{
		UserID:   req.GetUserId(),
		ChildID:  req.GetChildId(),
		Relation: protoRelationToString(req.GetRelation()),
	}

	err := s.guardianshipSvc.AddGuardian(ctx, dto)
	if err != nil {
		return nil, toGRPCError(err)
	}

	// 查询新创建的监护关系
	userID, _ := strconv.ParseUint(req.GetUserId(), 10, 64)
	childID, _ := strconv.ParseUint(req.GetChildId(), 10, 64)
	guardianship, err := s.guardRepo.FindByUserIDAndChildID(ctx, meta.FromUint64(userID), meta.FromUint64(childID))
	if err != nil {
		return &identityv1.AddGuardianResponse{}, nil
	}

	return &identityv1.AddGuardianResponse{
		Guardianship: guardianshipDomainToProto(guardianship),
	}, nil
}

// UpdateGuardianRelation 更新监护关系类型
func (s *guardianshipCommandServer) UpdateGuardianRelation(ctx context.Context, req *identityv1.UpdateGuardianRelationRequest) (*identityv1.UpdateGuardianRelationResponse, error) {
	if req == nil || strings.TrimSpace(req.GetGuardianshipId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "guardianship_id is required")
	}

	// 暂不支持更新关系类型
	return nil, status.Error(codes.Unimplemented, "update guardianship relation not implemented")
}

// RevokeGuardian 撤销监护关系
func (s *guardianshipCommandServer) RevokeGuardian(ctx context.Context, req *identityv1.RevokeGuardianRequest) (*identityv1.RevokeGuardianResponse, error) {
	if req == nil || req.GetTarget() == nil {
		return nil, status.Error(codes.InvalidArgument, "target is required")
	}

	var userID, childID string

	// 根据不同的 selector 解析
	switch target := req.GetTarget().GetSelector().(type) {
	case *identityv1.GuardianshipSelector_GuardianshipId:
		// 如果使用 guardianship_id，需要先查询得到 user_id 和 child_id
		guardianshipIDRaw := target.GuardianshipId
		guardianshipID, _ := strconv.ParseUint(guardianshipIDRaw, 10, 64)
		guardianship, err := s.guardRepo.FindByID(ctx, meta.FromUint64(guardianshipID))
		if err != nil {
			return nil, toGRPCError(err)
		}
		userID = guardianship.User.String()
		childID = guardianship.Child.String()

	case *identityv1.GuardianshipSelector_Key:
		userID = target.Key.GetUserId()
		childID = target.Key.GetChildId()

	default:
		return nil, status.Error(codes.InvalidArgument, "invalid target selector")
	}

	dto := guardianshipApp.RemoveGuardianDTO{
		UserID:  userID,
		ChildID: childID,
	}

	err := s.guardianshipSvc.RemoveGuardian(ctx, dto)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &identityv1.RevokeGuardianResponse{}, nil
}

// BatchRevokeGuardians 批量撤销监护关系
func (s *guardianshipCommandServer) BatchRevokeGuardians(ctx context.Context, req *identityv1.BatchRevokeGuardiansRequest) (*identityv1.BatchRevokeGuardiansResponse, error) {
	if req == nil || len(req.GetTargets()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "targets is required")
	}

	resp := &identityv1.BatchRevokeGuardiansResponse{
		Revoked:  make([]*identityv1.Guardianship, 0),
		Failures: make([]*identityv1.FailedGuardianshipFailure, 0),
	}

	for _, target := range req.GetTargets() {
		revokeReq := &identityv1.RevokeGuardianRequest{
			Target:   target,
			Reason:   req.GetReason(),
			Operator: req.GetOperator(),
		}

		_, err := s.RevokeGuardian(ctx, revokeReq)
		if err != nil {
			resp.Failures = append(resp.Failures, &identityv1.FailedGuardianshipFailure{
				Target: target,
				Error:  err.Error(),
			})
		}
	}

	return resp, nil
}

// ImportGuardians 批量导入监护关系
func (s *guardianshipCommandServer) ImportGuardians(ctx context.Context, req *identityv1.ImportGuardiansRequest) (*identityv1.ImportGuardiansResponse, error) {
	if req == nil || len(req.GetRecords()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "records is required")
	}

	resp := &identityv1.ImportGuardiansResponse{
		Created:  make([]*identityv1.Guardianship, 0),
		Failures: make([]*identityv1.FailedImportGuardian, 0),
	}

	for _, record := range req.GetRecords() {
		addReq := &identityv1.AddGuardianRequest{
			UserId:   record.GetUserId(),
			ChildId:  record.GetChildId(),
			Relation: record.GetRelation(),
			Operator: req.GetOperator(),
		}

		addResp, err := s.AddGuardian(ctx, addReq)
		if err != nil {
			resp.Failures = append(resp.Failures, &identityv1.FailedImportGuardian{
				Record: record,
				Error:  err.Error(),
			})
			continue
		}

		if addResp != nil && addResp.Guardianship != nil {
			resp.Created = append(resp.Created, addResp.Guardianship)
		}
	}

	return resp, nil
}

// ============= IdentityLifecycle 服务实现 =============

// CreateUser 创建用户
func (s *identityLifecycleServer) CreateUser(ctx context.Context, req *identityv1.CreateUserRequest) (*identityv1.CreateUserResponse, error) {
	if req == nil || strings.TrimSpace(req.GetNickname()) == "" {
		return nil, status.Error(codes.InvalidArgument, "nickname is required")
	}

	dto := userApp.RegisterUserDTO{
		Name:  req.GetNickname(),
		Phone: req.GetPhone(),
		Email: req.GetEmail(),
	}

	result, err := s.userSvc.Register(ctx, dto)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &identityv1.CreateUserResponse{
		User: userResultToProto(result),
	}, nil
}

// UpdateUser 更新用户
func (s *identityLifecycleServer) UpdateUser(ctx context.Context, req *identityv1.UpdateUserRequest) (*identityv1.UpdateUserResponse, error) {
	if req == nil || strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// 更新昵称
	if req.GetNickname() != "" {
		err := s.userProfileSvc.Rename(ctx, req.GetUserId(), req.GetNickname())
		if err != nil {
			return nil, toGRPCError(err)
		}
	}

	// 更新联系方式
	if req.GetPhone() != "" || req.GetEmail() != "" {
		dto := userApp.UpdateContactDTO{
			UserID: req.GetUserId(),
			Phone:  req.GetPhone(),
			Email:  req.GetEmail(),
		}
		err := s.userProfileSvc.UpdateContact(ctx, dto)
		if err != nil {
			return nil, toGRPCError(err)
		}
	}

	// 查询更新后的用户
	result, err := s.userSvc.Register(ctx, userApp.RegisterUserDTO{}) // 这里需要一个查询接口
	if err != nil {
		return &identityv1.UpdateUserResponse{}, nil
	}

	return &identityv1.UpdateUserResponse{
		User: userResultToProto(result),
	}, nil
}

// DeactivateUser 停用用户
func (s *identityLifecycleServer) DeactivateUser(ctx context.Context, req *identityv1.ChangeUserStatusRequest) (*identityv1.UserOperationResponse, error) {
	if req == nil || strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	err := s.userStatusSvc.Deactivate(ctx, req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &identityv1.UserOperationResponse{}, nil
}

// BlockUser 封禁用户
func (s *identityLifecycleServer) BlockUser(ctx context.Context, req *identityv1.ChangeUserStatusRequest) (*identityv1.UserOperationResponse, error) {
	if req == nil || strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	err := s.userStatusSvc.Block(ctx, req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &identityv1.UserOperationResponse{}, nil
}

// LinkExternalIdentity 绑定外部身份
func (s *identityLifecycleServer) LinkExternalIdentity(ctx context.Context, req *identityv1.LinkExternalIdentityRequest) (*identityv1.LinkExternalIdentityResponse, error) {
	if req == nil || strings.TrimSpace(req.GetUserId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// 暂不支持
	return nil, status.Error(codes.Unimplemented, "link external identity not implemented")
}

// ============= 辅助函数 =============

// protoRelationToString 将 proto 枚举转换为字符串
func protoRelationToString(relation identityv1.GuardianshipRelation) string {
	switch relation {
	case identityv1.GuardianshipRelation_GUARDIANSHIP_RELATION_SELF:
		return "self"
	case identityv1.GuardianshipRelation_GUARDIANSHIP_RELATION_PARENT:
		return "parent"
	case identityv1.GuardianshipRelation_GUARDIANSHIP_RELATION_GRANDPARENT:
		return "grandparent"
	case identityv1.GuardianshipRelation_GUARDIANSHIP_RELATION_OTHER:
		return "other"
	default:
		return "other"
	}
}

// stringToProtoRelation 将字符串转换为 proto 枚举
func stringToProtoRelation(relation string) identityv1.GuardianshipRelation {
	switch relation {
	case "self":
		return identityv1.GuardianshipRelation_GUARDIANSHIP_RELATION_SELF
	case "parent":
		return identityv1.GuardianshipRelation_GUARDIANSHIP_RELATION_PARENT
	case "grandparent", "grandparents":
		return identityv1.GuardianshipRelation_GUARDIANSHIP_RELATION_GRANDPARENT
	case "other":
		return identityv1.GuardianshipRelation_GUARDIANSHIP_RELATION_OTHER
	default:
		return identityv1.GuardianshipRelation_GUARDIANSHIP_RELATION_UNSPECIFIED
	}
}
