// Package identity 提供身份管理功能
package identity

import (
	"context"

	identityv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/identity/v1"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
)

// Client 身份服务客户端
type Client struct {
	readService      identityv1.IdentityReadClient
	lifecycleService identityv1.IdentityLifecycleClient
}

// NewClient 创建身份客户端
func NewClient(read identityv1.IdentityReadClient, lifecycle identityv1.IdentityLifecycleClient) *Client {
	return &Client{
		readService:      read,
		lifecycleService: lifecycle,
	}
}

// ========== 读取操作 ==========

// GetUser 获取用户信息
func (c *Client) GetUser(ctx context.Context, userID string) (*identityv1.GetUserResponse, error) {
	resp, err := c.readService.GetUser(ctx, &identityv1.GetUserRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// BatchGetUsers 批量获取用户
func (c *Client) BatchGetUsers(ctx context.Context, userIDs []string) (*identityv1.BatchGetUsersResponse, error) {
	resp, err := c.readService.BatchGetUsers(ctx, &identityv1.BatchGetUsersRequest{
		UserIds: userIDs,
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// SearchUsers 搜索用户
func (c *Client) SearchUsers(ctx context.Context, req *identityv1.SearchUsersRequest) (*identityv1.SearchUsersResponse, error) {
	resp, err := c.readService.SearchUsers(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// ========== 生命周期操作 ==========

// CreateUser 创建用户
func (c *Client) CreateUser(ctx context.Context, req *identityv1.CreateUserRequest) (*identityv1.CreateUserResponse, error) {
	resp, err := c.lifecycleService.CreateUser(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// UpdateUser 更新用户
func (c *Client) UpdateUser(ctx context.Context, req *identityv1.UpdateUserRequest) (*identityv1.UpdateUserResponse, error) {
	resp, err := c.lifecycleService.UpdateUser(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// DeactivateUser 停用用户
func (c *Client) DeactivateUser(ctx context.Context, req *identityv1.ChangeUserStatusRequest) (*identityv1.UserOperationResponse, error) {
	resp, err := c.lifecycleService.DeactivateUser(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// BlockUser 封禁用户
func (c *Client) BlockUser(ctx context.Context, req *identityv1.ChangeUserStatusRequest) (*identityv1.UserOperationResponse, error) {
	resp, err := c.lifecycleService.BlockUser(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// ========== 原始客户端 ==========

// ReadRaw 返回原始读取服务客户端
func (c *Client) ReadRaw() identityv1.IdentityReadClient {
	return c.readService
}

// LifecycleRaw 返回原始生命周期服务客户端
func (c *Client) LifecycleRaw() identityv1.IdentityLifecycleClient {
	return c.lifecycleService
}
