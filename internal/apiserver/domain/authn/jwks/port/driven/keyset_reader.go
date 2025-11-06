package driven

import (
	"context"
	"time"
)

type CacheTag struct {
	ETag         string
	LastModified time.Time
}

// 对外发布用：供应用层生成 /.well-known/jwks.json
type KeySetReader interface {
	CurrentJWKS(ctx context.Context) (jwksJSON []byte, tag CacheTag, err error)
	ActiveKeyMeta(ctx context.Context) (kid string, alg string, err error) // 给签名器或健康检查
}
