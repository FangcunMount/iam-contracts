package authz

import authzv4 "github.com/FangcunMount/iam/v5/api/grpc/iam/authz/v4"

// Raw 返回原始 AuthorizationService gRPC 客户端。
func (c *Client) Raw() authzv4.AuthorizationServiceClient {
	return c.authorizationService
}
