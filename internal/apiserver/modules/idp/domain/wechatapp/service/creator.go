package service

import (
	"context"
	"errors"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/idp/domain/wechatapp/port"
)

type WechatAppCreator struct {
	querier port.WechatAppQuerier
}

// 确保 WechatAppCreator 实现了 port.WechatAppCreator 接口
var _ port.WechatAppCreator = (*WechatAppCreator)(nil)

// NewWechatAppCreator 创建微信应用创建器
func NewWechatAppCreator(querier port.WechatAppQuerier) *WechatAppCreator {
	return &WechatAppCreator{querier: querier}
}

// Create 创建微信应用
func (c *WechatAppCreator) Create(ctx context.Context, appID string, name string, appType domain.AppType) (*domain.WechatApp, error) {
	// 参数校验
	if appID == "" {
		return nil, errors.New("appID cannot be empty")
	}
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	if appType == "" {
		return nil, errors.New("appType cannot be empty")
	}

	// 确定 appID 唯一性
	if exists, err := c.querier.ExistsByAppID(ctx, appID); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New("wechat app with the given appID already exists")
	}

	// 创建微信应用实体
	return domain.NewWechatApp(
		appType, appID,
		domain.WithWechatAppName(name),
		domain.WithWechatAppStatus(domain.StatusEnabled), // 默认启用
	), nil
}
