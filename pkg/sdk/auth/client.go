// Package auth 提供认证相关功能
package auth

import (
	"context"

	authnv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/authn/v1"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/errors"
)

// Client 认证服务客户端
type Client struct {
	authService authnv1.AuthServiceClient
	jwksService authnv1.JWKSServiceClient
}

// NewClient 创建认证客户端
func NewClient(authService authnv1.AuthServiceClient, jwksService authnv1.JWKSServiceClient) *Client {
	return &Client{
		authService: authService,
		jwksService: jwksService,
	}
}

// VerifyToken 验证 Token
func (c *Client) VerifyToken(ctx context.Context, req *authnv1.VerifyTokenRequest) (*authnv1.VerifyTokenResponse, error) {
	resp, err := c.authService.VerifyToken(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// RefreshToken 刷新 Token
func (c *Client) RefreshToken(ctx context.Context, req *authnv1.RefreshTokenRequest) (*authnv1.RefreshTokenResponse, error) {
	resp, err := c.authService.RefreshToken(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// RevokeToken 撤销 Token
func (c *Client) RevokeToken(ctx context.Context, req *authnv1.RevokeTokenRequest) (*authnv1.RevokeTokenResponse, error) {
	resp, err := c.authService.RevokeToken(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// RevokeRefreshToken 撤销 Refresh Token
func (c *Client) RevokeRefreshToken(ctx context.Context, req *authnv1.RevokeRefreshTokenRequest) (*authnv1.RevokeRefreshTokenResponse, error) {
	resp, err := c.authService.RevokeRefreshToken(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// IssueServiceToken 签发服务 Token
func (c *Client) IssueServiceToken(ctx context.Context, req *authnv1.IssueServiceTokenRequest) (*authnv1.IssueServiceTokenResponse, error) {
	resp, err := c.authService.IssueServiceToken(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// GetJWKS 获取 JWKS
func (c *Client) GetJWKS(ctx context.Context, req *authnv1.GetJWKSRequest) (*authnv1.GetJWKSResponse, error) {
	resp, err := c.jwksService.GetJWKS(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return resp, nil
}

// Raw 返回原始 gRPC 客户端
func (c *Client) Raw() authnv1.AuthServiceClient {
	return c.authService
}

// JWKSRaw 返回原始 JWKS gRPC 客户端
func (c *Client) JWKSRaw() authnv1.JWKSServiceClient {
	return c.jwksService
}
