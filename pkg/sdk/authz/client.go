// Package authz 提供授权判定（PDP）功能
package authz

import (
	"context"

	authzv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/authz/v1"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
)

// Client 授权服务客户端。
//
// 当前聚焦对外稳定的授权消费面，包括：
//   - 单次权限判定（Check）
//   - 便捷判定（Allow）
//   - 授权快照（GetAuthorizationSnapshot）
//   - 角色赋权/撤销（GrantAssignment / RevokeAssignment）
//
// 使用示例：
//
//	resp, err := client.Authz().Check(ctx, &authzv1.CheckRequest{
//	    Subject: "user:user-123",
//	    Domain:  "tenant-a",
//	    Object:  "resource:child_profile",
//	    Action:  "read",
//	})
//	if err != nil {
//	    return err
//	}
//	if resp.Allowed {
//	    fmt.Println("允许访问")
//	}
type Client struct {
	authorizationService authzv1.AuthorizationServiceClient
}

// NewClient 创建授权服务客户端。
func NewClient(authorizationService authzv1.AuthorizationServiceClient) *Client {
	return &Client{
		authorizationService: authorizationService,
	}
}

// Check 对单条 (subject, domain, object, action) 执行授权判定。
func (c *Client) Check(ctx context.Context, req *authzv1.CheckRequest) (*authzv1.CheckResponse, error) {
	resp, err := c.authorizationService.Check(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// Allow 是 Check 的便捷封装，只返回最终布尔结果。
func (c *Client) Allow(ctx context.Context, subject, domain, object, action string) (bool, error) {
	resp, err := c.Check(ctx, &authzv1.CheckRequest{
		Subject: subject,
		Domain:  domain,
		Object:  object,
		Action:  action,
	})
	if err != nil {
		return false, err
	}
	return resp.Allowed, nil
}

// GetAuthorizationSnapshot 获取主体在指定租户/应用下的授权快照。
func (c *Client) GetAuthorizationSnapshot(ctx context.Context, req *authzv1.GetAuthorizationSnapshotRequest) (*authzv1.GetAuthorizationSnapshotResponse, error) {
	resp, err := c.authorizationService.GetAuthorizationSnapshot(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// GrantAssignment 为主体授予角色。
func (c *Client) GrantAssignment(ctx context.Context, req *authzv1.GrantAssignmentRequest) (*authzv1.GrantAssignmentResponse, error) {
	resp, err := c.authorizationService.GrantAssignment(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// RevokeAssignment 撤销主体上的角色。
func (c *Client) RevokeAssignment(ctx context.Context, req *authzv1.RevokeAssignmentRequest) (*authzv1.RevokeAssignmentResponse, error) {
	resp, err := c.authorizationService.RevokeAssignment(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// Raw 返回原始 AuthorizationService gRPC 客户端。
func (c *Client) Raw() authzv1.AuthorizationServiceClient {
	return c.authorizationService
}
