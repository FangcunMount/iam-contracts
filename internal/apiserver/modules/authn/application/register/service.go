// Package register 注册应用服务
package register

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

// RegisterService 注册应用服务
// 负责协调不同类型的账号注册策略
type RegisterService struct {
	registerers map[string]Registerer // 注册器映射（按类型索引）
}

// NewRegisterService 创建注册应用服务
func NewRegisterService(registerers ...Registerer) *RegisterService {
	s := &RegisterService{
		registerers: make(map[string]Registerer),
	}

	// 注册所有提供的注册器
	for _, r := range registerers {
		if r != nil {
			s.registerers[r.Type()] = r
		}
	}

	return s
}

// Supports 检查是否支持指定类型的注册
func (s *RegisterService) Supports(registerType string) bool {
	_, exists := s.registerers[registerType]
	return exists
}

// RegisterWithWeChat 微信注册
func (s *RegisterService) RegisterWithWeChat(ctx context.Context, req *RegisterWithWeChatRequest) (*RegisterResponse, error) {
	if req == nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "request is required")
	}

	// 获取微信注册器
	registerer, exists := s.registerers["wechat"]
	if !exists {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "wechat registerer not found")
	}

	// 执行注册
	result, err := registerer.Register(ctx, req)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// RegisterWithWeChatRequest 微信注册请求
type RegisterWithWeChatRequest struct {
	Name     string            // 用户名
	Phone    string            // 手机号（必填）
	Email    string            // 邮箱（可选）
	AppID    string            // 微信应用ID
	OpenID   string            // 微信OpenID
	UnionID  *string           // 微信UnionID（可选）
	Nickname *string           // 微信昵称（可选）
	Avatar   *string           // 微信头像（可选）
	Meta     map[string]string // 微信元数据（可选）
}

// RegisterResponse 注册响应（统一）
type RegisterResponse struct {
	UserID    uint64 // 用户ID
	AccountID uint64 // 账号ID
}

// Registerer 注册器接口
// 不同类型的账号注册策略需要实现此接口
type Registerer interface {
	// Type 返回注册器类型（如 "wechat", "password" 等）
	Type() string

	// Register 执行注册
	Register(ctx context.Context, request interface{}) (*RegisterResponse, error)
}
