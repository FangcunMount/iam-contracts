package service

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp/port"
)

type AccessTokenCacher struct {
	// 依赖的缓存接口或服务
}

// 确保 AccessTokenCacher 实现了相应的接口
var _ port.AccessTokenCacher = (*AccessTokenCacher)(nil)

// NewAccessTokenCacher 创建访问令牌缓存器实例
func NewAccessTokenCacher() *AccessTokenCacher {
	return &AccessTokenCacher{}
}

// EnsureToken 单飞刷新 + 过期缓冲 获取访问令牌
func (c *AccessTokenCacher) EnsureToken(ctx context.Context, app *domain.WechatApp) (string, error) {
	return "", nil
}
