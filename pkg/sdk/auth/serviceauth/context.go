package serviceauth

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// AuthorizationContext 创建带 Authorization 头的 Context。
func AuthorizationContext(ctx context.Context, token string) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	md = md.Copy()
	md.Set("authorization", "Bearer "+token)
	return metadata.NewOutgoingContext(ctx, md)
}

// AuthorizationMetadata 返回包含 Authorization 的 metadata。
func AuthorizationMetadata(token string) metadata.MD {
	return metadata.Pairs("authorization", "Bearer "+token)
}
