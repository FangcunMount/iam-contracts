package wechatapp

import (
	"context"
	"fmt"
	"time"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/idp/wechatapp"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/idp/wechatapp/port"
)

// ============= 应用服务实现 =============

// ================================================
// ==== WechatAppApplicationService 实现 =====
// ================================================

type wechatAppApplicationService struct {
	repo    port.WechatAppRepository
	creator port.WechatAppCreator
	querier port.WechatAppQuerier
	rotater port.CredentialRotater
}

// NewWechatAppApplicationService 创建微信应用管理应用服务
func NewWechatAppApplicationService(
	repo port.WechatAppRepository,
	creator port.WechatAppCreator,
	querier port.WechatAppQuerier,
	rotater port.CredentialRotater,
) WechatAppApplicationService {
	return &wechatAppApplicationService{
		repo:    repo,
		creator: creator,
		querier: querier,
		rotater: rotater,
	}
}

// CreateApp 创建微信应用
func (s *wechatAppApplicationService) CreateApp(ctx context.Context, dto CreateWechatAppDTO) (*WechatAppResult, error) {
	// 调用领域服务创建微信应用实体
	app, err := s.creator.Create(ctx, dto.AppID, dto.Name, dto.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to create wechat app: %w", err)
	}

	// 分配内部 ID
	app.ID = idutil.NewID(idutil.GetIntID())

	// 初始化凭据结构
	app.Cred = &domain.Credentials{}

	// 如果提供了 AppSecret，设置认证密钥
	if dto.AppSecret != "" {
		if err := s.rotater.RotateAuthSecret(ctx, app, dto.AppSecret); err != nil {
			return nil, fmt.Errorf("failed to set auth secret: %w", err)
		}
	}

	// 持久化
	if err := s.repo.Create(ctx, app); err != nil {
		return nil, fmt.Errorf("failed to persist wechat app: %w", err)
	}

	// 转换为结果 DTO
	return toWechatAppResult(app), nil
}

// GetApp 查询微信应用
func (s *wechatAppApplicationService) GetApp(ctx context.Context, appID string) (*WechatAppResult, error) {
	app, err := s.querier.QueryByAppID(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to query wechat app: %w", err)
	}

	if app == nil {
		return nil, fmt.Errorf("wechat app not found: %s", appID)
	}

	return toWechatAppResult(app), nil
}

// =========================================================
// ==== WechatAppCredentialApplicationService 实现 =====
// =========================================================

type wechatAppCredentialApplicationService struct {
	repo    port.WechatAppRepository
	querier port.WechatAppQuerier
	rotater port.CredentialRotater
}

// NewWechatAppCredentialApplicationService 创建微信应用凭据应用服务
func NewWechatAppCredentialApplicationService(
	repo port.WechatAppRepository,
	querier port.WechatAppQuerier,
	rotater port.CredentialRotater,
) WechatAppCredentialApplicationService {
	return &wechatAppCredentialApplicationService{
		repo:    repo,
		querier: querier,
		rotater: rotater,
	}
}

// RotateAuthSecret 轮换认证密钥（AppSecret）
func (s *wechatAppCredentialApplicationService) RotateAuthSecret(ctx context.Context, appID string, newSecret string) error {
	// 查询应用
	app, err := s.querier.QueryByAppID(ctx, appID)
	if err != nil {
		return fmt.Errorf("failed to query wechat app: %w", err)
	}
	if app == nil {
		return fmt.Errorf("wechat app not found: %s", appID)
	}

	// 确保凭据结构已初始化
	if app.Cred == nil {
		app.Cred = &domain.Credentials{}
	}

	// 调用领域服务轮换密钥
	if err := s.rotater.RotateAuthSecret(ctx, app, newSecret); err != nil {
		return fmt.Errorf("failed to rotate auth secret: %w", err)
	}

	// 持久化
	if err := s.repo.Update(ctx, app); err != nil {
		return fmt.Errorf("failed to update wechat app: %w", err)
	}

	return nil
}

// RotateMsgSecret 轮换消息加解密密钥
func (s *wechatAppCredentialApplicationService) RotateMsgSecret(ctx context.Context, appID string, callbackToken string, encodingAESKey string) error {
	// 查询应用
	app, err := s.querier.QueryByAppID(ctx, appID)
	if err != nil {
		return fmt.Errorf("failed to query wechat app: %w", err)
	}
	if app == nil {
		return fmt.Errorf("wechat app not found: %s", appID)
	}

	// 确保凭据结构已初始化
	if app.Cred == nil {
		app.Cred = &domain.Credentials{}
	}

	// 调用领域服务轮换密钥
	if err := s.rotater.RotateMsgAESKey(ctx, app, callbackToken, encodingAESKey); err != nil {
		return fmt.Errorf("failed to rotate msg secret: %w", err)
	}

	// 持久化
	if err := s.repo.Update(ctx, app); err != nil {
		return fmt.Errorf("failed to update wechat app: %w", err)
	}

	return nil
}

// ======================================================
// ==== WechatAppTokenApplicationService 实现 =====
// ======================================================

type wechatAppTokenApplicationService struct {
	querier       port.WechatAppQuerier
	tokenCacher   port.AccessTokenCacher
	tokenProvider port.AppTokenProvider
	cache         port.AccessTokenCache
}

// NewWechatAppTokenApplicationService 创建微信应用访问令牌应用服务
func NewWechatAppTokenApplicationService(
	querier port.WechatAppQuerier,
	tokenCacher port.AccessTokenCacher,
	tokenProvider port.AppTokenProvider,
	cache port.AccessTokenCache,
) WechatAppTokenApplicationService {
	return &wechatAppTokenApplicationService{
		querier:       querier,
		tokenCacher:   tokenCacher,
		tokenProvider: tokenProvider,
		cache:         cache,
	}
}

// GetAccessToken 获取访问令牌（带缓存和自动刷新）
func (s *wechatAppTokenApplicationService) GetAccessToken(ctx context.Context, appID string) (string, error) {
	// 查询应用
	app, err := s.querier.QueryByAppID(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("failed to query wechat app: %w", err)
	}
	if app == nil {
		return "", fmt.Errorf("wechat app not found: %s", appID)
	}

	// 使用缓存器获取令牌（自动处理刷新）
	token, err := s.tokenCacher.EnsureToken(ctx, app, 120*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	return token, nil
}

// RefreshAccessToken 强制刷新访问令牌
func (s *wechatAppTokenApplicationService) RefreshAccessToken(ctx context.Context, appID string) (string, error) {
	// 查询应用
	app, err := s.querier.QueryByAppID(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("failed to query wechat app: %w", err)
	}
	if app == nil {
		return "", fmt.Errorf("wechat app not found: %s", appID)
	}

	// 强制刷新：获取新令牌
	aat, err := s.tokenProvider.Fetch(ctx, app)
	if err != nil {
		return "", fmt.Errorf("failed to fetch access token: %w", err)
	}

	// 更新缓存
	ttl := time.Until(aat.ExpiresAt)
	if ttl < 60*time.Second {
		ttl = 60 * time.Second
	}
	if err := s.cache.Set(ctx, appID, aat, ttl); err != nil {
		return "", fmt.Errorf("failed to cache access token: %w", err)
	}

	return aat.Token, nil
}

// ============= 辅助函数 =============

// toWechatAppResult 转换领域对象为结果 DTO
func toWechatAppResult(app *domain.WechatApp) *WechatAppResult {
	if app == nil {
		return nil
	}

	return &WechatAppResult{
		ID:     app.ID.String(),
		AppID:  app.AppID,
		Name:   app.Name,
		Type:   app.Type,
		Status: app.Status,
	}
}
