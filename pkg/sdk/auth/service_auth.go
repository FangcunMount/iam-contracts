package auth

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	authnv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/authn/v1"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/config"
	"google.golang.org/grpc/metadata"
)

// =============================================================================
// 刷新策略配置
// =============================================================================

// RefreshStrategy 刷新策略配置
type RefreshStrategy struct {
	// JitterRatio 刷新时间抖动比例（0-1），默认 0.1 表示 ±10%
	JitterRatio float64

	// MinBackoff 失败后最小退避时间
	MinBackoff time.Duration

	// MaxBackoff 失败后最大退避时间
	MaxBackoff time.Duration

	// BackoffMultiplier 退避乘数
	BackoffMultiplier float64

	// MaxRetries 最大连续重试次数，超过后进入熔断
	MaxRetries int

	// CircuitOpenDuration 熔断持续时间
	CircuitOpenDuration time.Duration

	// OnRefreshSuccess 刷新成功回调
	OnRefreshSuccess func(token string, expiresIn time.Duration)

	// OnRefreshFailure 刷新失败回调
	OnRefreshFailure func(err error, attempt int, nextRetry time.Duration)

	// OnCircuitOpen 熔断打开回调
	OnCircuitOpen func()

	// OnCircuitClose 熔断关闭回调
	OnCircuitClose func()
}

// DefaultRefreshStrategy 默认刷新策略
func DefaultRefreshStrategy() *RefreshStrategy {
	return &RefreshStrategy{
		JitterRatio:         0.1,
		MinBackoff:          1 * time.Second,
		MaxBackoff:          60 * time.Second,
		BackoffMultiplier:   2.0,
		MaxRetries:          5,
		CircuitOpenDuration: 30 * time.Second,
	}
}

// =============================================================================
// 刷新状态
// =============================================================================

// RefreshState 刷新状态
type RefreshState int32

const (
	RefreshStateNormal      RefreshState = iota // 正常刷新
	RefreshStateRetrying                        // 重试中
	RefreshStateCircuitOpen                     // 熔断中
)

func (s RefreshState) String() string {
	switch s {
	case RefreshStateNormal:
		return "normal"
	case RefreshStateRetrying:
		return "retrying"
	case RefreshStateCircuitOpen:
		return "circuit_open"
	default:
		return "unknown"
	}
}

// RefreshStats 刷新统计
type RefreshStats struct {
	TotalRefreshes      int64
	SuccessfulRefreshes int64
	FailedRefreshes     int64
	ConsecutiveFailures int64
	LastRefreshTime     time.Time
	LastRefreshError    error
	State               RefreshState
}

// =============================================================================
// ServiceAuthHelper 增强版
// =============================================================================

// ServiceAuthHelper 服务间认证助手
// 用于简化服务间 Token 的获取和传递
type ServiceAuthHelper struct {
	config     *config.ServiceAuthConfig
	authClient *Client
	strategy   *RefreshStrategy

	mu           sync.RWMutex
	currentToken string
	expiresAt    time.Time
	stopCh       chan struct{}

	// 刷新状态
	state               atomic.Int32
	consecutiveFailures atomic.Int64
	circuitOpenUntil    time.Time

	// 统计
	stats struct {
		totalRefreshes      atomic.Int64
		successfulRefreshes atomic.Int64
		failedRefreshes     atomic.Int64
		lastRefreshTime     atomic.Value // time.Time
		lastRefreshError    atomic.Value // error
	}
}

// ServiceAuthOption 配置选项
type ServiceAuthOption func(*ServiceAuthHelper)

// WithRefreshStrategy 设置刷新策略
func WithRefreshStrategy(strategy *RefreshStrategy) ServiceAuthOption {
	return func(h *ServiceAuthHelper) {
		h.strategy = strategy
	}
}

// NewServiceAuthHelper 创建服务认证助手
func NewServiceAuthHelper(cfg *config.ServiceAuthConfig, authClient *Client, opts ...ServiceAuthOption) (*ServiceAuthHelper, error) {
	h := &ServiceAuthHelper{
		config:     cfg,
		authClient: authClient,
		strategy:   DefaultRefreshStrategy(),
		stopCh:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(h)
	}

	// 立即获取一次 token
	if err := h.refreshTokenWithRetry(context.Background()); err != nil {
		return nil, err
	}

	// 启动后台刷新
	go h.refreshLoop()

	return h, nil
}

// GetToken 获取当前有效的服务 Token
func (h *ServiceAuthHelper) GetToken(ctx context.Context) (string, error) {
	h.mu.RLock()
	token := h.currentToken
	expiresAt := h.expiresAt
	h.mu.RUnlock()

	// 检查熔断状态
	if RefreshState(h.state.Load()) == RefreshStateCircuitOpen {
		// 熔断中，如果还有有效 token 则继续使用
		if token != "" && time.Now().Before(expiresAt) {
			return token, nil
		}
		return "", fmt.Errorf("service_auth: circuit breaker open, no valid token available")
	}

	// 检查是否需要刷新
	if time.Until(expiresAt) < h.config.RefreshBefore {
		if err := h.refreshTokenWithRetry(ctx); err != nil {
			// 如果刷新失败但还有有效 token，继续使用
			if token != "" && time.Now().Before(expiresAt) {
				return token, nil
			}
			return "", err
		}

		h.mu.RLock()
		token = h.currentToken
		h.mu.RUnlock()
	}

	return token, nil
}

// NewAuthenticatedContext 创建带认证信息的 Context
func (h *ServiceAuthHelper) NewAuthenticatedContext(ctx context.Context) (context.Context, error) {
	token, err := h.GetToken(ctx)
	if err != nil {
		return nil, err
	}
	return AuthorizationContext(ctx, token), nil
}

// CallWithAuth 使用认证信息执行调用
func (h *ServiceAuthHelper) CallWithAuth(ctx context.Context, fn func(ctx context.Context) error) error {
	authCtx, err := h.NewAuthenticatedContext(ctx)
	if err != nil {
		return err
	}
	return fn(authCtx)
}

// Stop 停止后台刷新
func (h *ServiceAuthHelper) Stop() {
	close(h.stopCh)
}

// Stats 获取刷新统计
func (h *ServiceAuthHelper) Stats() RefreshStats {
	stats := RefreshStats{
		TotalRefreshes:      h.stats.totalRefreshes.Load(),
		SuccessfulRefreshes: h.stats.successfulRefreshes.Load(),
		FailedRefreshes:     h.stats.failedRefreshes.Load(),
		ConsecutiveFailures: h.consecutiveFailures.Load(),
		State:               RefreshState(h.state.Load()),
	}

	if t := h.stats.lastRefreshTime.Load(); t != nil {
		stats.LastRefreshTime = t.(time.Time)
	}
	if e := h.stats.lastRefreshError.Load(); e != nil {
		stats.LastRefreshError = e.(error)
	}

	return stats
}

// State 获取当前刷新状态
func (h *ServiceAuthHelper) State() RefreshState {
	return RefreshState(h.state.Load())
}

// =============================================================================
// 内部方法
// =============================================================================

func (h *ServiceAuthHelper) refreshLoop() {
	for {
		// 计算下次刷新时间
		nextRefresh := h.calculateNextRefresh()

		select {
		case <-time.After(nextRefresh):
			h.refreshTokenWithRetry(context.Background())
		case <-h.stopCh:
			return
		}
	}
}

func (h *ServiceAuthHelper) calculateNextRefresh() time.Duration {
	h.mu.RLock()
	expiresAt := h.expiresAt
	h.mu.RUnlock()

	// 基础刷新时间
	refreshInterval := time.Until(expiresAt) - h.config.RefreshBefore
	if refreshInterval <= 0 {
		refreshInterval = h.config.TokenTTL / 2
	}

	// 添加 jitter
	jitter := h.addJitter(refreshInterval)

	// 如果在重试状态，使用退避时间
	if RefreshState(h.state.Load()) == RefreshStateRetrying {
		backoff := h.calculateBackoff(int(h.consecutiveFailures.Load()))
		if backoff < jitter {
			return backoff
		}
	}

	// 如果在熔断状态，等待熔断结束
	if RefreshState(h.state.Load()) == RefreshStateCircuitOpen {
		h.mu.RLock()
		waitUntil := h.circuitOpenUntil
		h.mu.RUnlock()
		wait := time.Until(waitUntil)
		if wait > 0 {
			return wait
		}
	}

	return jitter
}

func (h *ServiceAuthHelper) addJitter(d time.Duration) time.Duration {
	if h.strategy.JitterRatio <= 0 {
		return d
	}

	// 计算抖动范围：±JitterRatio
	jitterRange := float64(d) * h.strategy.JitterRatio
	jitter := jitterRange * (rand.Float64()*2 - 1)

	return time.Duration(float64(d) + jitter)
}

func (h *ServiceAuthHelper) calculateBackoff(failures int) time.Duration {
	if failures <= 0 {
		return h.strategy.MinBackoff
	}

	// 指数退避
	backoff := float64(h.strategy.MinBackoff) * math.Pow(h.strategy.BackoffMultiplier, float64(failures-1))

	// 限制最大值
	if backoff > float64(h.strategy.MaxBackoff) {
		backoff = float64(h.strategy.MaxBackoff)
	}

	// 添加抖动
	return h.addJitter(time.Duration(backoff))
}

func (h *ServiceAuthHelper) refreshTokenWithRetry(ctx context.Context) error {
	// 检查熔断状态
	if RefreshState(h.state.Load()) == RefreshStateCircuitOpen {
		h.mu.RLock()
		openUntil := h.circuitOpenUntil
		h.mu.RUnlock()

		if time.Now().Before(openUntil) {
			return fmt.Errorf("service_auth: circuit breaker open until %v", openUntil)
		}

		// 熔断时间已过，尝试恢复
		h.closeCircuit()
	}

	h.stats.totalRefreshes.Add(1)

	err := h.refreshToken(ctx)
	if err != nil {
		return h.handleRefreshFailure(err)
	}

	return h.handleRefreshSuccess()
}

func (h *ServiceAuthHelper) handleRefreshSuccess() error {
	h.stats.successfulRefreshes.Add(1)
	h.stats.lastRefreshTime.Store(time.Now())
	h.consecutiveFailures.Store(0)
	h.state.Store(int32(RefreshStateNormal))

	// 回调
	if h.strategy.OnRefreshSuccess != nil {
		h.mu.RLock()
		token := h.currentToken
		expiresAt := h.expiresAt
		h.mu.RUnlock()
		h.strategy.OnRefreshSuccess(token, time.Until(expiresAt))
	}

	return nil
}

func (h *ServiceAuthHelper) handleRefreshFailure(err error) error {
	h.stats.failedRefreshes.Add(1)
	h.stats.lastRefreshError.Store(err)
	failures := h.consecutiveFailures.Add(1)

	// 计算下次重试时间
	nextRetry := h.calculateBackoff(int(failures))

	// 回调
	if h.strategy.OnRefreshFailure != nil {
		h.strategy.OnRefreshFailure(err, int(failures), nextRetry)
	}

	// 检查是否需要熔断
	if int(failures) >= h.strategy.MaxRetries {
		h.openCircuit()
		return fmt.Errorf("service_auth: circuit breaker opened after %d failures: %w", failures, err)
	}

	// 进入重试状态
	h.state.Store(int32(RefreshStateRetrying))

	return err
}

func (h *ServiceAuthHelper) openCircuit() {
	h.state.Store(int32(RefreshStateCircuitOpen))

	h.mu.Lock()
	h.circuitOpenUntil = time.Now().Add(h.strategy.CircuitOpenDuration)
	h.mu.Unlock()

	if h.strategy.OnCircuitOpen != nil {
		h.strategy.OnCircuitOpen()
	}
}

func (h *ServiceAuthHelper) closeCircuit() {
	oldState := RefreshState(h.state.Swap(int32(RefreshStateNormal)))

	if oldState == RefreshStateCircuitOpen {
		h.consecutiveFailures.Store(0)
		if h.strategy.OnCircuitClose != nil {
			h.strategy.OnCircuitClose()
		}
	}
}

func (h *ServiceAuthHelper) refreshToken(ctx context.Context) error {
	resp, err := h.authClient.IssueServiceToken(ctx, &authnv1.IssueServiceTokenRequest{
		Subject:  h.config.ServiceID,
		Audience: h.config.TargetAudience,
	})
	if err != nil {
		return err
	}

	// TokenPair 包含 access_token
	tokenPair := resp.TokenPair
	if tokenPair == nil {
		return fmt.Errorf("service_auth: empty token pair in response")
	}

	// 计算过期时间
	expiresIn := tokenPair.ExpiresIn.AsDuration()

	h.mu.Lock()
	h.currentToken = tokenPair.AccessToken
	h.expiresAt = time.Now().Add(expiresIn)
	h.mu.Unlock()

	return nil
}

// =============================================================================
// 便捷构造函数
// =============================================================================

// NewServiceAuthHelperWithCallbacks 创建带回调的服务认证助手
func NewServiceAuthHelperWithCallbacks(
	cfg *config.ServiceAuthConfig,
	authClient *Client,
	onSuccess func(token string, expiresIn time.Duration),
	onFailure func(err error, attempt int, nextRetry time.Duration),
) (*ServiceAuthHelper, error) {
	strategy := DefaultRefreshStrategy()
	strategy.OnRefreshSuccess = onSuccess
	strategy.OnRefreshFailure = onFailure

	return NewServiceAuthHelper(cfg, authClient, WithRefreshStrategy(strategy))
}

// =============================================================================
// Context 工具函数
// =============================================================================

// AuthorizationContext 创建带 Authorization 头的 Context
func AuthorizationContext(ctx context.Context, token string) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	md = md.Copy()
	md.Set("authorization", "Bearer "+token)
	return metadata.NewOutgoingContext(ctx, md)
}

// AuthorizationMetadata 返回包含 Authorization 的 metadata
func AuthorizationMetadata(token string) metadata.MD {
	return metadata.Pairs("authorization", "Bearer "+token)
}
