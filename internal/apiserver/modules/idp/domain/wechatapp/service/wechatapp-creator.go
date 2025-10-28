package service

import (
	"context"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp/port"
)

type WechatAppCreator struct {
	repo port.WechatAppRepository
}

// 确保 WechatAppCreator 实现了 port.WechatAppCreator 接口
var _ port.WechatAppCreator = (*WechatAppCreator)(nil)

// NewWechatAppCreator 创建微信应用创建器
func NewWechatAppCreator(repo port.WechatAppRepository) *WechatAppCreator {
	return &WechatAppCreator{repo: repo}
}

// Create 创建微信应用
func (c *WechatAppCreator) Create(ctx context.Context, appID string, name string, appType domain.AppType) (*domain.WechatApp, error) {

	return &domain.WechatApp{}, nil
}
