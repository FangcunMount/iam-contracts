package authnsdk

import (
	"context"
	"fmt"

	authnv1 "github.com/FangcunMount/iam-contracts/internal/apiserver/interface/authn/grpc/pb/iam/authn/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps AuthService gRPC client.
type Client struct {
	conn       *grpc.ClientConn
	authClient authnv1.AuthServiceClient
}

// NewClient dials IAM authn gRPC endpoint. If dialOptions is empty, insecure transport is used.
func NewClient(ctx context.Context, endpoint string, dialOptions ...grpc.DialOption) (*Client, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("grpc endpoint is empty")
	}
	opts := dialOptions
	if len(opts) == 0 {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
	conn, err := grpc.DialContext(ctx, endpoint, opts...)
	if err != nil {
		return nil, err
	}
	client := authnv1.NewAuthServiceClient(conn)
	return &Client{
		conn:       conn,
		authClient: client,
	}, nil
}

// Close closes underlying connection.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) Auth() authnv1.AuthServiceClient {
	return c.authClient
}
