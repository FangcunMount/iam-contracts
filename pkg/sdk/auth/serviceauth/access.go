package serviceauth

import (
	"context"
	"fmt"
	"time"
)

// GetToken 获取当前有效的服务 Token。
func (h *ServiceAuthHelper) GetToken(ctx context.Context) (string, error) {
	h.mu.RLock()
	token := h.currentToken
	expiresAt := h.expiresAt
	h.mu.RUnlock()

	if RefreshState(h.state.Load()) == RefreshStateCircuitOpen {
		if token != "" && time.Now().Before(expiresAt) {
			return token, nil
		}
		return "", fmt.Errorf("service_auth: circuit breaker open, no valid token available")
	}

	if time.Until(expiresAt) < h.config.RefreshBefore {
		if err := h.refreshTokenWithRetry(ctx); err != nil {
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

// NewAuthenticatedContext 创建带认证信息的 Context。
func (h *ServiceAuthHelper) NewAuthenticatedContext(ctx context.Context) (context.Context, error) {
	token, err := h.GetToken(ctx)
	if err != nil {
		return nil, err
	}
	return AuthorizationContext(ctx, token), nil
}

// CallWithAuth 使用认证信息执行调用。
func (h *ServiceAuthHelper) CallWithAuth(ctx context.Context, fn func(ctx context.Context) error) error {
	authCtx, err := h.NewAuthenticatedContext(ctx)
	if err != nil {
		return err
	}
	return fn(authCtx)
}

// Stop 停止后台刷新。
func (h *ServiceAuthHelper) Stop() {
	close(h.stopCh)
}

// Stats 获取刷新统计。
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

// State 获取当前刷新状态。
func (h *ServiceAuthHelper) State() RefreshState {
	return RefreshState(h.state.Load())
}
