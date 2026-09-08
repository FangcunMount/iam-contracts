// Package authz 提供授权判定（PDP）能力。
package authz

import authzv4 "github.com/FangcunMount/iam/v5/api/grpc/iam/authz/v4"

// Client 授权服务客户端。
type Client struct {
	authorizationService authzv4.AuthorizationServiceClient
}

// NewClient 创建授权服务客户端。
func NewClient(authorizationService authzv4.AuthorizationServiceClient) *Client {
	return &Client{
		authorizationService: authorizationService,
	}
}
