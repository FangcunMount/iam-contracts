package identity

import (
	"context"

	identityv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/identity/v1"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
)

// GuardianshipClient 监护关系服务客户端
type GuardianshipClient struct {
	queryService   identityv1.GuardianshipQueryClient
	commandService identityv1.GuardianshipCommandClient
}

// NewGuardianshipClient 创建监护关系客户端
func NewGuardianshipClient(query identityv1.GuardianshipQueryClient, command identityv1.GuardianshipCommandClient) *GuardianshipClient {
	return &GuardianshipClient{
		queryService:   query,
		commandService: command,
	}
}

// ========== 查询操作 ==========

// IsGuardian 检查是否是监护人
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

// ListChildren 列出用户的所有被监护儿童
func (c *GuardianshipClient) ListChildren(ctx context.Context, req *identityv1.ListChildrenRequest) (*identityv1.ListChildrenResponse, error) {
	resp, err := c.queryService.ListChildren(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// ListGuardians 列出儿童的所有监护人
func (c *GuardianshipClient) ListGuardians(ctx context.Context, req *identityv1.ListGuardiansRequest) (*identityv1.ListGuardiansResponse, error) {
	resp, err := c.queryService.ListGuardians(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// ========== 命令操作 ==========

// AddGuardian 添加监护关系
func (c *GuardianshipClient) AddGuardian(ctx context.Context, req *identityv1.AddGuardianRequest) (*identityv1.AddGuardianResponse, error) {
	resp, err := c.commandService.AddGuardian(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// RevokeGuardian 撤销监护关系
func (c *GuardianshipClient) RevokeGuardian(ctx context.Context, req *identityv1.RevokeGuardianRequest) (*identityv1.RevokeGuardianResponse, error) {
	resp, err := c.commandService.RevokeGuardian(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// UpdateGuardianRelation 更新监护关系
func (c *GuardianshipClient) UpdateGuardianRelation(ctx context.Context, req *identityv1.UpdateGuardianRelationRequest) (*identityv1.UpdateGuardianRelationResponse, error) {
	resp, err := c.commandService.UpdateGuardianRelation(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// BatchRevokeGuardians 批量撤销监护关系
func (c *GuardianshipClient) BatchRevokeGuardians(ctx context.Context, req *identityv1.BatchRevokeGuardiansRequest) (*identityv1.BatchRevokeGuardiansResponse, error) {
	resp, err := c.commandService.BatchRevokeGuardians(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// ImportGuardians 批量导入监护关系
func (c *GuardianshipClient) ImportGuardians(ctx context.Context, req *identityv1.ImportGuardiansRequest) (*identityv1.ImportGuardiansResponse, error) {
	resp, err := c.commandService.ImportGuardians(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// ========== 原始客户端 ==========

// QueryRaw 返回原始查询服务客户端
func (c *GuardianshipClient) QueryRaw() identityv1.GuardianshipQueryClient {
	return c.queryService
}

// CommandRaw 返回原始命令服务客户端
func (c *GuardianshipClient) CommandRaw() identityv1.GuardianshipCommandClient {
	return c.commandService
}
