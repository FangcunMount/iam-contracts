package wechatapp

import (
	"context"
	"errors"
)

type creator struct {
	repo Repository
}

// 确保 creator 实现了 Creator 接口
var _ Creator = (*creator)(nil)

// NewCreator 创建微信应用创建器
func NewCreator(repo Repository) Creator {
	return &creator{repo: repo}
}

// Create 创建微信应用
func (c *creator) Create(ctx context.Context, appID string, name string, appType AppType) (*WechatApp, error) {
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
	existing, err := c.repo.GetByAppID(ctx, appID)
	if err == nil && existing != nil {
		return nil, errors.New("wechat app with the given appID already exists")
	}

	// 创建微信应用实体
	return NewWechatApp(
		appType, appID,
		WithWechatAppName(name),
		WithWechatAppStatus(StatusEnabled), // 默认启用
	), nil
}
