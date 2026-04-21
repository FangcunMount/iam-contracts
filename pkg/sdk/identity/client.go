// Package identity 提供身份管理功能
package identity

import (
	"context"

	identityv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/identity/v1"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
)

// Client 身份服务客户端。
//
// 提供用户和孩子身份的管理功能，包括：
//   - 读取操作（GetUser、BatchGetUsers、SearchUsers、GetChild、BatchGetChildren）
//   - 生命周期操作（CreateUser、UpdateUser、DeactivateUser、BlockUser）
//
// 使用示例：
//
//	resp, err := client.Identity().GetUser(ctx, "user-123")
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("用户: %s, 状态: %s\n", resp.GetUser().GetNickname(), resp.GetUser().GetStatus().String())
type Client struct {
	readService      identityv1.IdentityReadClient
	lifecycleService identityv1.IdentityLifecycleClient
}

// NewClient 创建身份服务客户端。
//
// 参数：
//   - read: gRPC 读取服务客户端
//   - lifecycle: gRPC 生命周期服务客户端
//
// 返回：
//   - *Client: 身份服务客户端实例
func NewClient(read identityv1.IdentityReadClient, lifecycle identityv1.IdentityLifecycleClient) *Client {
	return &Client{
		readService:      read,
		lifecycleService: lifecycle,
	}
}

// ========== 读取操作 ==========

// GetUser 获取用户信息。
//
// 参数：
//   - ctx: 请求上下文
//   - userID: 用户 ID
//
// 返回：
//   - *identityv1.GetUserResponse: 包含 User 对象，常用字段有 nickname、status、contacts
//   - error: 可能的错误类型包括 InvalidArgument、NotFound、PermissionDenied
//
// 示例：
//
//	resp, err := client.Identity().GetUser(ctx, "user-123")
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("用户: %s (%s)\n", resp.GetUser().GetNickname(), resp.GetUser().GetStatus().String())
func (c *Client) GetUser(ctx context.Context, userID string) (*identityv1.GetUserResponse, error) {
	resp, err := c.readService.GetUser(ctx, &identityv1.GetUserRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// BatchGetUsers 批量获取用户信息。
//
// 一次请求获取多个用户的信息，比多次调用 GetUser 更高效。
//
// 参数：
//   - ctx: 请求上下文
//   - userIDs: 用户 ID 列表
//
// 返回：
//   - *identityv1.BatchGetUsersResponse: 包含 Users（用户列表）和 NotFoundIds（未找到的 ID）
//   - error: 可能的错误类型包括 InvalidArgument、PermissionDenied
//
// 示例：
//
//	resp, err := client.Identity().BatchGetUsers(ctx, []string{"user-1", "user-2", "user-3"})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("找到 %d 个用户\n", len(resp.Users))
//	if len(resp.NotFoundIds) > 0 {
//	    fmt.Printf("未找到: %v\n", resp.NotFoundIds)
//	}
func (c *Client) BatchGetUsers(ctx context.Context, userIDs []string) (*identityv1.BatchGetUsersResponse, error) {
	resp, err := c.readService.BatchGetUsers(ctx, &identityv1.BatchGetUsersRequest{
		UserIds: userIDs,
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// SearchUsers 搜索用户。
//
// 根据关键词、手机号或邮箱搜索用户，支持偏移分页。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 搜索请求，包含 Keyword、Phones、Emails、Page 等字段
//
// 返回：
//   - *identityv1.SearchUsersResponse: 包含 Users、Total、Page 字段
//   - error: 可能的错误类型包括 InvalidArgument、PermissionDenied
//
// 示例：
//
//	resp, err := client.Identity().SearchUsers(ctx, &identityv1.SearchUsersRequest{
//	    Keyword: "张三",
//	    Page:    &identityv1.OffsetPagination{Limit: 20, Offset: 0},
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("找到 %d 个用户\n", resp.Total)
func (c *Client) SearchUsers(ctx context.Context, req *identityv1.SearchUsersRequest) (*identityv1.SearchUsersResponse, error) {
	resp, err := c.readService.SearchUsers(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// GetChild 获取孩子信息。
//
// 参数：
//   - ctx: 请求上下文
//   - childID: 孩子 ID
//
// 返回：
//   - *identityv1.GetChildResponse: 包含 Child 对象，常用字段有 legal_name、dob、gender
//   - error: 可能的错误类型包括 InvalidArgument、NotFound、PermissionDenied
//
// 示例：
//
//	resp, err := client.Identity().GetChild(ctx, "child-456")
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("孩子: %s, 生日: %s\n", resp.GetChild().GetLegalName(), resp.GetChild().GetDob())
func (c *Client) GetChild(ctx context.Context, childID string) (*identityv1.GetChildResponse, error) {
	resp, err := c.readService.GetChild(ctx, &identityv1.GetChildRequest{
		ChildId: childID,
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// BatchGetChildren 批量获取孩子信息。
//
// 一次请求获取多个孩子的信息，比多次调用 GetChild 更高效。
//
// 参数：
//   - ctx: 请求上下文
//   - childIDs: 孩子 ID 列表
//
// 返回：
//   - *identityv1.BatchGetChildrenResponse: 包含 Children（孩子列表）和 NotFound（未找到的 ID）
//   - error: 可能的错误类型包括 InvalidArgument、PermissionDenied
//
// 示例：
//
//	resp, err := client.Identity().BatchGetChildren(ctx, []string{"child-1", "child-2", "child-3"})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("找到 %d 个孩子\n", len(resp.Children))
func (c *Client) BatchGetChildren(ctx context.Context, childIDs []string) (*identityv1.BatchGetChildrenResponse, error) {
	resp, err := c.readService.BatchGetChildren(ctx, &identityv1.BatchGetChildrenRequest{
		ChildIds: childIDs,
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// ========== 生命周期操作 ==========

// CreateUser 创建用户。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 创建请求，包含 Nickname、Phone、Email 等字段
//
// 返回：
//   - *identityv1.CreateUserResponse: 包含新创建的 User 对象
//   - error: 可能的错误类型包括 InvalidArgument、AlreadyExists、PermissionDenied
//
// 示例：
//
//	resp, err := client.Identity().CreateUser(ctx, &identityv1.CreateUserRequest{
//	    Nickname: "张三",
//	    Phone:    "13800138000",
//	    Email:    "zhangsan@example.com",
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("用户创建成功，ID: %s\n", resp.GetUser().GetId())
func (c *Client) CreateUser(ctx context.Context, req *identityv1.CreateUserRequest) (*identityv1.CreateUserResponse, error) {
	resp, err := c.lifecycleService.CreateUser(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// UpdateUser 更新用户信息。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 更新请求，包含 UserId、Nickname、Phone、Email 等字段
//
// 返回：
//   - *identityv1.UpdateUserResponse: 包含更新后的 User 对象
//   - error: 可能的错误类型包括 InvalidArgument、NotFound、PermissionDenied
//
// 示例：
//
//	resp, err := client.Identity().UpdateUser(ctx, &identityv1.UpdateUserRequest{
//	    UserId:   "user-123",
//	    Nickname: "李四",
//	    Phone:    "13800138001",
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("用户信息更新成功\n")
func (c *Client) UpdateUser(ctx context.Context, req *identityv1.UpdateUserRequest) (*identityv1.UpdateUserResponse, error) {
	resp, err := c.lifecycleService.UpdateUser(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// DeactivateUser 停用用户。
//
// 将用户设置为停用状态，停用后用户无法登录但可以恢复。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 状态变更请求，包含 UserId、Reason 等字段
//
// 返回：
//   - *identityv1.UserOperationResponse: 包含变更后的 User 对象
//   - error: 可能的错误类型包括 InvalidArgument、NotFound、PermissionDenied
//
// 示例：
//
//	resp, err := client.Identity().DeactivateUser(ctx, &identityv1.ChangeUserStatusRequest{
//	    UserId: "user-123",
//	    Reason: "用户申请注销",
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("用户停用后状态: %s\n", resp.GetUser().GetStatus().String())
func (c *Client) DeactivateUser(ctx context.Context, req *identityv1.ChangeUserStatusRequest) (*identityv1.UserOperationResponse, error) {
	resp, err := c.lifecycleService.DeactivateUser(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// BlockUser 封禁用户。
//
// 将用户设置为封禁状态，封禁后用户无法访问任何资源。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 状态变更请求，包含 UserId、Reason 等字段
//
// 返回：
//   - *identityv1.UserOperationResponse: 包含变更后的 User 对象
//   - error: 可能的错误类型包括 InvalidArgument、NotFound、PermissionDenied
//
// 示例：
//
//	resp, err := client.Identity().BlockUser(ctx, &identityv1.ChangeUserStatusRequest{
//	    UserId: "user-123",
//	    Reason: "违反社区规则",
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("用户封禁后状态: %s\n", resp.GetUser().GetStatus().String())
func (c *Client) BlockUser(ctx context.Context, req *identityv1.ChangeUserStatusRequest) (*identityv1.UserOperationResponse, error) {
	resp, err := c.lifecycleService.BlockUser(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// ========== 原始客户端 ==========

// ReadRaw 返回原始读取服务客户端。
//
// 用于访问 SDK 未封装的原始 gRPC 方法。
//
// 返回：
//   - identityv1.IdentityReadClient: 原始 gRPC 读取客户端
func (c *Client) ReadRaw() identityv1.IdentityReadClient {
	return c.readService
}

// LifecycleRaw 返回原始生命周期服务客户端。
//
// 用于访问 SDK 未封装的原始 gRPC 方法。
//
// 返回：
//   - identityv1.IdentityLifecycleClient: 原始 gRPC 生命周期客户端
func (c *Client) LifecycleRaw() identityv1.IdentityLifecycleClient {
	return c.lifecycleService
}
