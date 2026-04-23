package serviceauth

import (
	"context"
	"math"
	"math/rand"
	"time"
)

func (h *ServiceAuthHelper) refreshLoop() {
	for {
		nextRefresh := h.calculateNextRefresh()

		select {
		case <-time.After(nextRefresh):
			_ = h.refreshTokenWithRetry(context.Background())
		case <-h.stopCh:
			return
		}
	}
}

func (h *ServiceAuthHelper) calculateNextRefresh() time.Duration {
	h.mu.RLock()
	expiresAt := h.expiresAt
	h.mu.RUnlock()

	refreshInterval := time.Until(expiresAt) - h.config.RefreshBefore
	if refreshInterval <= 0 {
		refreshInterval = h.config.TokenTTL / 2
	}

	jitter := h.addJitter(refreshInterval)

	if RefreshState(h.state.Load()) == RefreshStateRetrying {
		backoff := h.calculateBackoff(int(h.consecutiveFailures.Load()))
		if backoff < jitter {
			return backoff
		}
	}

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

	jitterRange := float64(d) * h.strategy.JitterRatio
	jitter := jitterRange * (rand.Float64()*2 - 1)
	return time.Duration(float64(d) + jitter)
}

func (h *ServiceAuthHelper) calculateBackoff(failures int) time.Duration {
	if failures <= 0 {
		return h.strategy.MinBackoff
	}

	backoff := float64(h.strategy.MinBackoff) * math.Pow(h.strategy.BackoffMultiplier, float64(failures-1))
	if backoff > float64(h.strategy.MaxBackoff) {
		backoff = float64(h.strategy.MaxBackoff)
	}
	return h.addJitter(time.Duration(backoff))
}
