package verifier

import (
	"context"
	"fmt"
)

// Verify 验证 Token。
func (v *TokenVerifier) Verify(ctx context.Context, token string, opts *VerifyOptions) (*VerifyResult, error) {
	if opts == nil {
		opts = &VerifyOptions{}
	}
	if opts.ForceRemote {
		if v.remoteStrategy == nil {
			return nil, fmt.Errorf("token verifier: remote strategy not available")
		}
		return v.remoteStrategy.Verify(ctx, token, opts)
	}
	return v.strategy.Verify(ctx, token, opts)
}

// Strategy 返回当前使用的策略。
func (v *TokenVerifier) Strategy() VerifyStrategy {
	return v.strategy
}
