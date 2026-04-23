package serviceauth

import "time"

// DefaultRefreshStrategy 默认刷新策略。
func DefaultRefreshStrategy() *RefreshStrategy {
	return &RefreshStrategy{
		JitterRatio:         0.1,
		MinBackoff:          time.Second,
		MaxBackoff:          60 * time.Second,
		BackoffMultiplier:   2.0,
		MaxRetries:          5,
		CircuitOpenDuration: 30 * time.Second,
	}
}

// WithRefreshStrategy 设置刷新策略。
func WithRefreshStrategy(strategy *RefreshStrategy) ServiceAuthOption {
	return func(h *ServiceAuthHelper) {
		h.strategy = strategy
	}
}

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
