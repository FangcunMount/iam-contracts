// Package idp 提供身份提供者（IDP）功能
package idp

import (
	"context"

	idpv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/idp/v1"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
)

// Client IDP 服务客户端。
//
// 提供微信应用管理功能，包括：
//   - 查询操作（GetWechatApp）
//
// 使用示例：
//
//	resp, err := client.IDP().GetWechatApp(ctx, "wx1234567890")
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("微信应用: %s, 名称: %s\n", resp.App.AppId, resp.App.Name)
type Client struct {
	idpService idpv1.IDPServiceClient
}

// NewClient 创建 IDP 服务客户端。
//
// 参数：
//   - idp: gRPC IDP 服务客户端
//
// 返回：
//   - *Client: IDP 服务客户端实例
func NewClient(idp idpv1.IDPServiceClient) *Client {
	return &Client{
		idpService: idp,
	}
}

// ========== 读取操作 ==========

// GetWechatApp 根据 AppID 查询微信应用。
//
// 参数：
//   - ctx: 请求上下文
//   - appID: 微信应用 ID
//
// 返回：
//   - *idpv1.GetWechatAppResponse: 包含 WechatApp 对象，包含 AppID、名称、类型、状态等信息
//   - error: 可能的错误类型包括 InvalidArgument、NotFound、PermissionDenied
//
// 示例：
//
//	resp, err := client.IDP().GetWechatApp(ctx, "wx1234567890")
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("微信应用: %s (%s)\n", resp.App.Name, resp.App.AppId)
func (c *Client) GetWechatApp(ctx context.Context, appID string) (*idpv1.GetWechatAppResponse, error) {
	resp, err := c.idpService.GetWechatApp(ctx, &idpv1.GetWechatAppRequest{
		AppId: appID,
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// ========== 原始客户端 ==========

// Raw 返回原始 IDP 服务客户端。
//
// 用于访问 SDK 未封装的原始 gRPC 方法。
//
// 返回：
//   - idpv1.IDPServiceClient: 原始 gRPC IDP 客户端
func (c *Client) Raw() idpv1.IDPServiceClient {
	return c.idpService
}
