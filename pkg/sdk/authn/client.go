package authnsdk

import (
	"context"
	"fmt"

	"github.com/FangcunMount/component-base/pkg/log"
	authnv1 "github.com/FangcunMount/iam-contracts/internal/apiserver/interface/authn/grpc/pb/iam/authn/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client 认证服务 gRPC 客户端
// 封装了与 IAM 认证服务的 gRPC 连接和调用
type Client struct {
	conn       *grpc.ClientConn          // gRPC 连接
	authClient authnv1.AuthServiceClient // 认证服务客户端
}

// NewClient 创建认证服务客户端
// 连接到 IAM 认证服务的 gRPC 端点
//
// 参数：
//   - ctx: 上下文
//   - endpoint: gRPC 服务地址，格式为 "host:port"
//   - dialOptions: gRPC 连接选项（可选），如果为空则使用不安全连接
//
// 返回：
//   - *Client: 客户端实例
//   - error: 连接失败时返回错误
func NewClient(ctx context.Context, endpoint string, dialOptions ...grpc.DialOption) (*Client, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("grpc endpoint is empty")
	}
	log.Infof("[AuthN SDK] Connecting to IAM authn gRPC endpoint: %s", endpoint)
	opts := dialOptions
	if len(opts) == 0 {
		log.Debug("[AuthN SDK] Using insecure credentials for gRPC connection")
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
	conn, err := grpc.DialContext(ctx, endpoint, opts...)
	if err != nil {
		log.Errorf("[AuthN SDK] Failed to connect to gRPC endpoint %s: %v", endpoint, err)
		return nil, err
	}
	log.Infof("[AuthN SDK] Successfully connected to IAM authn gRPC endpoint")
	client := authnv1.NewAuthServiceClient(conn)
	return &Client{
		conn:       conn,
		authClient: client,
	}, nil
}

// Close 关闭 gRPC 连接
// 释放底层连接资源
//
// 返回：
//   - error: 关闭失败时返回错误
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	log.Debug("[AuthN SDK] Closing gRPC connection")
	err := c.conn.Close()
	if err != nil {
		log.Warnf("[AuthN SDK] Error closing gRPC connection: %v", err)
	} else {
		log.Debug("[AuthN SDK] gRPC connection closed successfully")
	}
	return err
}

// Auth 获取认证服务客户端
// 返回底层的 gRPC 认证服务客户端，可用于调用认证相关的 RPC 方法
//
// 返回：
//   - AuthServiceClient: 认证服务客户端接口
func (c *Client) Auth() authnv1.AuthServiceClient {
	return c.authClient
}
