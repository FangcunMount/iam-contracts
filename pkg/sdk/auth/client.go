// Package auth 提供认证相关功能
package auth

import (
	"context"

	authnv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/authn/v1"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
)

// Client 认证服务客户端。
//
// 提供认证相关功能，包括：
//   - Token 验证和管理（VerifyToken、RefreshToken、RevokeToken、RevokeRefreshToken）
//   - 服务间认证（IssueServiceToken）
//   - JWKS 管理（GetJWKS）
//
// 使用示例：
//
//	resp, err := client.Auth().VerifyToken(ctx, &authnv1.VerifyTokenRequest{
//	    Token: "eyJhbGciOiJSUzI1NiIs...",
//	})
//	if err != nil {
//	    return err
//	}
//	if resp.Valid {
//	    fmt.Printf("用户ID: %s\n", resp.UserId)
//	}
type Client struct {
	authService authnv1.AuthServiceClient
	jwksService authnv1.JWKSServiceClient
}

// NewClient 创建认证服务客户端。
//
// 参数：
//   - authService: gRPC 认证服务客户端
//   - jwksService: gRPC JWKS 服务客户端
//
// 返回：
//   - *Client: 认证服务客户端实例
func NewClient(authService authnv1.AuthServiceClient, jwksService authnv1.JWKSServiceClient) *Client {
	return &Client{
		authService: authService,
		jwksService: jwksService,
	}
}

// VerifyToken 验证 Access Token 的有效性。
//
// 检查 Token 的签名、过期时间、颁发者等信息。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 验证请求，包含 Token 字段
//
// 返回：
//   - *authnv1.VerifyTokenResponse: 包含 Valid（布尔值）、UserId、Claims 字段
//   - error: 可能的错误类型包括 InvalidArgument、Unauthenticated
//
// 示例：
//
//	resp, err := client.Auth().VerifyToken(ctx, &authnv1.VerifyTokenRequest{
//	    Token: "eyJhbGciOiJSUzI1NiIs...",
//	})
//	if err != nil {
//	    return err
//	}
//	if resp.Valid {
//	    fmt.Printf("用户ID: %s, 角色: %v\n", resp.UserId, resp.Claims["roles"])
//	} else {
//	    fmt.Printf("Token 无效: %s\n", resp.Reason)
//	}
func (c *Client) VerifyToken(ctx context.Context, req *authnv1.VerifyTokenRequest) (*authnv1.VerifyTokenResponse, error) {
	resp, err := c.authService.VerifyToken(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// RefreshToken 使用 Refresh Token 刷新获取新的 Access Token。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 刷新请求，包含 RefreshToken 字段
//
// 返回：
//   - *authnv1.RefreshTokenResponse: 包含 AccessToken、RefreshToken（可选）、ExpiresIn 字段
//   - error: 可能的错误类型包括 InvalidArgument、Unauthenticated
//
// 示例：
//
//	resp, err := client.Auth().RefreshToken(ctx, &authnv1.RefreshTokenRequest{
//	    RefreshToken: "refresh_token_abc123",
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("新 Token: %s, 过期时间: %d 秒\n", resp.AccessToken, resp.ExpiresIn)
func (c *Client) RefreshToken(ctx context.Context, req *authnv1.RefreshTokenRequest) (*authnv1.RefreshTokenResponse, error) {
	resp, err := c.authService.RefreshToken(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// RevokeToken 撤销 Access Token。
//
// 立即失效指定的 Access Token，常用于登出操作。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 撤销请求，包含 Token 字段
//
// 返回：
//   - *authnv1.RevokeTokenResponse: 包含 Success 和 Message 字段
//   - error: 可能的错误类型包括 InvalidArgument、NotFound
//
// 示例：
//
//	resp, err := client.Auth().RevokeToken(ctx, &authnv1.RevokeTokenRequest{
//	    Token: "eyJhbGciOiJSUzI1NiIs...",
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Token 撤销结果: %s\n", resp.Message)
func (c *Client) RevokeToken(ctx context.Context, req *authnv1.RevokeTokenRequest) (*authnv1.RevokeTokenResponse, error) {
	resp, err := c.authService.RevokeToken(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// RevokeRefreshToken 撤销 Refresh Token。
//
// 立即失效指定的 Refresh Token，防止用户使用该 Token 刷新 Access Token。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 撤销请求，包含 RefreshToken 字段
//
// 返回：
//   - *authnv1.RevokeRefreshTokenResponse: 包含 Success 和 Message 字段
//   - error: 可能的错误类型包括 InvalidArgument、NotFound
//
// 示例：
//
//	resp, err := client.Auth().RevokeRefreshToken(ctx, &authnv1.RevokeRefreshTokenRequest{
//	    RefreshToken: "refresh_token_abc123",
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Refresh Token 撤销成功\n")
func (c *Client) RevokeRefreshToken(ctx context.Context, req *authnv1.RevokeRefreshTokenRequest) (*authnv1.RevokeRefreshTokenResponse, error) {
	resp, err := c.authService.RevokeRefreshToken(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// IssueServiceToken 签发服务间认证 Token。
//
// 用于服务之间的相互认证，支持限制访问范围。
//
// 参数：
//   - ctx: 请求上下文
//   - req: 签发请求，包含 ServiceId、Audience、ExpiresIn 等字段
//
// 返回：
//   - *authnv1.IssueServiceTokenResponse: 包含 Token、TokenType、ExpiresIn 字段
//   - error: 可能的错误类型包括 InvalidArgument、PermissionDenied
//
// 示例：
//
//	resp, err := client.Auth().IssueServiceToken(ctx, &authnv1.IssueServiceTokenRequest{
//	    ServiceId: "service-api",
//	    Audience:  "service-backend",
//	    ExpiresIn: 3600,
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("服务 Token: %s\n", resp.Token)
func (c *Client) IssueServiceToken(ctx context.Context, req *authnv1.IssueServiceTokenRequest) (*authnv1.IssueServiceTokenResponse, error) {
	resp, err := c.authService.IssueServiceToken(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// GetJWKS 获取 JSON Web Key Set (JWKS)。
//
// 返回用于验证 JWT 签名的公钥集合。
//
// 参数：
//   - ctx: 请求上下文
//   - req: JWKS 请求，通常为空
//
// 返回：
//   - *authnv1.GetJWKSResponse: 包含 Keys（公钥列表）字段
//   - error: 可能的错误类型包括 Internal
//
// 示例：
//
//	resp, err := client.Auth().GetJWKS(ctx, &authnv1.GetJWKSRequest{})
//	if err != nil {
//	    return err
//	}
//	for _, key := range resp.Keys {
//	    fmt.Printf("Key ID: %s, Algorithm: %s\n", key.Kid, key.Alg)
//	}
func (c *Client) GetJWKS(ctx context.Context, req *authnv1.GetJWKSRequest) (*authnv1.GetJWKSResponse, error) {
	resp, err := c.jwksService.GetJWKS(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// Raw 返回原始认证服务 gRPC 客户端。
//
// 用于访问 SDK 未封装的原始 gRPC 方法。
//
// 返回：
//   - authnv1.AuthServiceClient: 原始 gRPC 认证服务客户端
func (c *Client) Raw() authnv1.AuthServiceClient {
	return c.authService
}

// JWKSRaw 返回原始 JWKS 服务 gRPC 客户端。
//
// 用于访问 SDK 未封装的原始 gRPC 方法。
//
// 返回：
//   - authnv1.JWKSServiceClient: 原始 gRPC JWKS 服务客户端
func (c *Client) JWKSRaw() authnv1.JWKSServiceClient {
	return c.jwksService
}
