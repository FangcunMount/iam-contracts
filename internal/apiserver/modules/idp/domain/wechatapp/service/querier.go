package service

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp/port"
)

// WechatAppQuerier 微信应用查询器
type WechatAppQuerier struct {
	// 依赖的仓储接口
	repo port.WechatAppRepository
}

// 确保 WechatAppQuerier 实现了 port.WechatAppQuerier 接口
var _ port.WechatAppQuerier = (*WechatAppQuerier)(nil)

// NewWechatAppQuerier 创建微信应用查询器
func NewWechatAppQuerier(repo port.WechatAppRepository) *WechatAppQuerier {
	return &WechatAppQuerier{repo: repo}
}

// GetByAppID 根据 AppID 查询微信应用
func (f *WechatAppQuerier) QueryByAppID(ctx context.Context, appID string) (*domain.WechatApp, error) {
	return f.repo.GetByAppID(ctx, appID)
}

// ExistsByAppID 检查微信应用是否存在
func (f *WechatAppQuerier) ExistsByAppID(ctx context.Context, appID string) (bool, error) {
	app, err := f.repo.GetByAppID(ctx, appID)
	if err != nil {
		return false, err
	}

	isExists := app != nil
	return isExists, nil
}
