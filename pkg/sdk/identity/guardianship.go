package identity

import (
	"context"

	identityv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/identity/v1"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
)

// GuardianshipClient 监护关系服务客户端。
//
// 提供监护关系的查询和管理功能，包括：
//   - 查询监护关系（IsGuardian、ListChildren、ListGuardians）
//   - 管理监护关系（AddGuardian、UpdateGuardianRelation、RevokeGuardian）
//   - 批量操作（BatchRevokeGuardians、ImportGuardians）
//
// 使用示例：
//
//	resp, err := client.Guardianship().IsGuardian(ctx, "user-123", "child-456")
//	if err != nil {
//	    return err
//	}
//	if resp.IsGuardian {
//	    fmt.Println("是监护人")
//	}
type GuardianshipClient struct {
	queryService   identityv1.GuardianshipQueryClient
	commandService identityv1.GuardianshipCommandClient
}

// NewGuardianshipClient 创建监护关系客户端。
//
// 参数：
//   - query: gRPC 查询服务客户端
//   - command: gRPC 命令服务客户端
//
// 返回：
//   - *GuardianshipClient: 监护关系客户端实例
func NewGuardianshipClient(query identityv1.GuardianshipQueryClient, command identityv1.GuardianshipCommandClient) *GuardianshipClient {
	return &GuardianshipClient{
		queryService:   query,
		commandService: command,
	}
}

// ========== 查询操作 ==========

// IsGuardian 检查用户是否是孩子的监护人。
//
// 参数：
//   - ctx: 请求上下文
//   - userID: 用户 ID
//   - childID: 孩子 ID
//
// 返回：
//   - *identityv1.IsGuardianResponse: 包含 IsGuardian（布尔值）和 Relation（关系详情）字段
//   - error: 可能的错误类型包括 InvalidArgument、NotFound、PermissionDenied
//
// 示例：
//
//	resp, err := client.Guardianship().IsGuardian(ctx, "user-123", "child-456")
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("是监护人: %v, 关系: %s\n", resp.IsGuardian, resp.Relation.RelationType)
func (c *GuardianshipClient) IsGuardian(ctx context.Context, userID, childID string) (*identityv1.IsGuardianResponse, error) {
	resp, err := c.queryService.IsGuardian(ctx, &identityv1.IsGuardianRequest{
		UserId:  userID,
		ChildId: childID,
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// ListChildren 列出用户的所有被监护儿童。
//
// 支持分页查询和过滤条件。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 列表请求，包含 UserId、Page、PageSize、Status 等字段
//
// 返回：
//   - *identityv1.ListChildrenResponse: 包含 Children（儿童列表）、Total、Page、PageSize 字段
//   - error: 可能的错误类型包括 InvalidArgument、NotFound、PermissionDenied
//
// 示例：
//
//	resp, err := client.Guardianship().ListChildren(ctx, &identityv1.ListChildrenRequest{
//	    UserId:   "user-123",
//	    Page:     1,
//	    PageSize: 20,
//	    Status:   "active",
//	})
//	if err != nil {
//	    return err
//	}
//	for _, child := range resp.Children {
//	    fmt.Printf("孩子: %s, 关系: %s\n", child.Name, child.RelationType)
//	}
func (c *GuardianshipClient) ListChildren(ctx context.Context, req *identityv1.ListChildrenRequest) (*identityv1.ListChildrenResponse, error) {
	resp, err := c.queryService.ListChildren(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// GetUserChildren 获取用户的监护孩子列表（便捷方法）。
//
// 这是 ListChildren 的便捷封装，使用默认分页参数，适合快速获取所有儿童。
//
// 参数：
//   - ctx: 请求上下文
//   - userID: 用户 ID
//
// 返回：
//   - *identityv1.ListChildrenResponse: 包含 Children、Total、Page、PageSize 字段
//   - error: 可能的错误类型包括 InvalidArgument、NotFound、PermissionDenied
//
// 示例：
//
//	resp, err := client.Guardianship().GetUserChildren(ctx, "user-123")
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("共有 %d 个孩子\n", resp.Total)
//	for _, child := range resp.Children {
//	    fmt.Printf("- %s (ID: %s)\n", child.Name, child.ChildId)
//	}
func (c *GuardianshipClient) GetUserChildren(ctx context.Context, userID string) (*identityv1.ListChildrenResponse, error) {
	resp, err := c.queryService.ListChildren(ctx, &identityv1.ListChildrenRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// ListGuardians 列出儿童的所有监护人。
//
// 支持分页查询和过滤条件。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 列表请求，包含 ChildId、Page、PageSize、Status 等字段
//
// 返回：
//   - *identityv1.ListGuardiansResponse: 包含 Guardians（监护人列表）、Total、Page、PageSize 字段
//   - error: 可能的错误类型包括 InvalidArgument、NotFound、PermissionDenied
//
// 示例：
//
//	resp, err := client.Guardianship().ListGuardians(ctx, &identityv1.ListGuardiansRequest{
//	    ChildId:  "child-456",
//	    Page:     1,
//	    PageSize: 20,
//	})
//	if err != nil {
//	    return err
//	}
//	for _, guardian := range resp.Guardians {
//	    fmt.Printf("监护人: %s, 关系: %s\n", guardian.Name, guardian.RelationType)
//	}
func (c *GuardianshipClient) ListGuardians(ctx context.Context, req *identityv1.ListGuardiansRequest) (*identityv1.ListGuardiansResponse, error) {
	resp, err := c.queryService.ListGuardians(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// ========== 命令操作 ==========

// AddGuardian 添加监护关系。
//
// 为孩子添加新的监护人，支持多种关系类型（父亲、母亲、祖父母等）。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 添加请求，包含 UserId、ChildId、RelationType、Metadata 等字段
//
// 返回：
//   - *identityv1.AddGuardianResponse: 包含 RelationId（新建关系的 ID）和 Success 字段
//   - error: 可能的错误类型包括 InvalidArgument、AlreadyExists、PermissionDenied
//
// 示例：
//
//	resp, err := client.Guardianship().AddGuardian(ctx, &identityv1.AddGuardianRequest{
//	    UserId:       "user-123",
//	    ChildId:      "child-456",
//	    RelationType: "father",
//	    Metadata:     map[string]string{"verified": "true"},
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("监护关系创建成功，关系ID: %s\n", resp.RelationId)
func (c *GuardianshipClient) AddGuardian(ctx context.Context, req *identityv1.AddGuardianRequest) (*identityv1.AddGuardianResponse, error) {
	resp, err := c.commandService.AddGuardian(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// RevokeGuardian 撤销监护关系。
//
// 移除用户与孩子之间的监护关系。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 撤销请求，包含 UserId、ChildId、Reason 等字段
//
// 返回：
//   - *identityv1.RevokeGuardianResponse: 包含 Success 和 Message 字段
//   - error: 可能的错误类型包括 InvalidArgument、NotFound、PermissionDenied
//
// 示例：
//
//	resp, err := client.Guardianship().RevokeGuardian(ctx, &identityv1.RevokeGuardianRequest{
//	    UserId:  "user-123",
//	    ChildId: "child-456",
//	    Reason:  "用户申请解除监护关系",
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("撤销结果: %s\n", resp.Message)
func (c *GuardianshipClient) RevokeGuardian(ctx context.Context, req *identityv1.RevokeGuardianRequest) (*identityv1.RevokeGuardianResponse, error) {
	resp, err := c.commandService.RevokeGuardian(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// UpdateGuardianRelation 更新监护关系。
//
// 修改现有监护关系的信息，如关系类型、元数据等。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 更新请求，包含 RelationId、RelationType、Metadata、UpdateMask 等字段
//
// 返回：
//   - *identityv1.UpdateGuardianRelationResponse: 包含更新后的 Relation 对象
//   - error: 可能的错误类型包括 InvalidArgument、NotFound、PermissionDenied
//
// 示例：
//
//	resp, err := client.Guardianship().UpdateGuardianRelation(ctx, &identityv1.UpdateGuardianRelationRequest{
//	    RelationId:   "rel-789",
//	    RelationType: "stepfather",
//	    Metadata:     map[string]string{"verified": "true", "updated_at": "2024-01-01"},
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("关系更新成功: %+v\n", resp.Relation)
func (c *GuardianshipClient) UpdateGuardianRelation(ctx context.Context, req *identityv1.UpdateGuardianRelationRequest) (*identityv1.UpdateGuardianRelationResponse, error) {
	resp, err := c.commandService.UpdateGuardianRelation(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// BatchRevokeGuardians 批量撤销监护关系。
//
// 一次性撤销多个监护关系，支持部分成功。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 批量撤销请求，包含 RelationIds（关系 ID 列表）和 Reason 字段
//
// 返回：
//   - *identityv1.BatchRevokeGuardiansResponse: 包含 SuccessCount、FailureCount、Results（详细结果）字段
//   - error: 如果整个请求失败则返回错误，部分失败不返回错误
//
// 示例：
//
//	resp, err := client.Guardianship().BatchRevokeGuardians(ctx, &identityv1.BatchRevokeGuardiansRequest{
//	    RelationIds: []string{"rel-1", "rel-2", "rel-3"},
//	    Reason:      "批量清理过期监护关系",
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("成功: %d, 失败: %d\n", resp.SuccessCount, resp.FailureCount)
//	for _, result := range resp.Results {
//	    if !result.Success {
//	        fmt.Printf("关系 %s 撤销失败: %s\n", result.RelationId, result.Error)
//	    }
//	}
func (c *GuardianshipClient) BatchRevokeGuardians(ctx context.Context, req *identityv1.BatchRevokeGuardiansRequest) (*identityv1.BatchRevokeGuardiansResponse, error) {
	resp, err := c.commandService.BatchRevokeGuardians(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// ImportGuardians 批量导入监护关系。
//
// 一次性导入多个监护关系，适用于数据迁移或初始化场景，支持部分成功。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 批量导入请求，包含 Relations（关系列表）和 SkipDuplicates 字段
//
// 返回：
//   - *identityv1.ImportGuardiansResponse: 包含 SuccessCount、FailureCount、SkippedCount、Results 字段
//   - error: 如果整个请求失败则返回错误，部分失败不返回错误
//
// 示例：
//
//	resp, err := client.Guardianship().ImportGuardians(ctx, &identityv1.ImportGuardiansRequest{
//	    Relations: []*identityv1.GuardianRelationInput{
//	        {UserId: "user-1", ChildId: "child-1", RelationType: "father"},
//	        {UserId: "user-2", ChildId: "child-2", RelationType: "mother"},
//	    },
//	    SkipDuplicates: true,
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("成功: %d, 失败: %d, 跳过: %d\n", resp.SuccessCount, resp.FailureCount, resp.SkippedCount)
func (c *GuardianshipClient) ImportGuardians(ctx context.Context, req *identityv1.ImportGuardiansRequest) (*identityv1.ImportGuardiansResponse, error) {
	resp, err := c.commandService.ImportGuardians(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// ========== 原始客户端 ==========

// QueryRaw 返回原始查询服务客户端。
//
// 用于访问 SDK 未封装的原始 gRPC 方法。
//
// 返回：
//   - identityv1.GuardianshipQueryClient: 原始 gRPC 查询客户端
func (c *GuardianshipClient) QueryRaw() identityv1.GuardianshipQueryClient {
	return c.queryService
}

// CommandRaw 返回原始命令服务客户端。
//
// 用于访问 SDK 未封装的原始 gRPC 方法。
//
// 返回：
//   - identityv1.GuardianshipCommandClient: 原始 gRPC 命令客户端
func (c *GuardianshipClient) CommandRaw() identityv1.GuardianshipCommandClient {
	return c.commandService
}
