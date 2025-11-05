package authentication

import (
	"context"
	"fmt"

	authPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port"
	tokenPort "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/token/port"
)

// TokenVerifierAdapter JWT令牌验证适配器
// 将 token 模块的 TokenVerifier 适配为 authentication 模块需要的接口
type TokenVerifierAdapter struct {
	tokenVerifier tokenPort.TokenVerifier
}

// 实现 authentication.port.TokenVerifier 接口
var _ authPort.TokenVerifier = (*TokenVerifierAdapter)(nil)

// NewTokenVerifierAdapter 创建令牌验证适配器
func NewTokenVerifierAdapter(tokenVerifier tokenPort.TokenVerifier) *TokenVerifierAdapter {
	return &TokenVerifierAdapter{
		tokenVerifier: tokenVerifier,
	}
}

// VerifyAccessToken 验证访问令牌
// 从 token 模块的 TokenClaims 中提取用户ID、账户ID等信息
func (a *TokenVerifierAdapter) VerifyAccessToken(ctx context.Context, tokenValue string) (userID, accountID int64, tenantID *int64, err error) {
	// 调用 token 模块的验证服务
	claims, err := a.tokenVerifier.VerifyAccessToken(ctx, tokenValue)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("token verification failed: %w", err)
	}

	// 从 claims 中提取信息
	userID = int64(claims.UserID.ToUint64())
	accountID = int64(claims.AccountID.ToUint64())

	// tenantID 目前从 token claims 中无法获取，返回 nil
	// 如果需要，可以在 TokenClaims 中添加 TenantID 字段
	return userID, accountID, nil, nil
}
