package authentication

import (
	"context"
	"fmt"

	authPort "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication"
	tokenPort "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/token"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// TokenVerifierAdapter JWT令牌验证适配器
// 将 token 模块的 TokenVerifier 适配为 authentication 模块需要的接口
type TokenVerifierAdapter struct {
	tokenVerifier tokenPort.Verifier
}

// 实现 authentication.wechatapp.TokenVerifier 接口
var _ authPort.TokenVerifier = (*TokenVerifierAdapter)(nil)

// NewTokenVerifierAdapter 创建令牌验证适配器
func NewTokenVerifierAdapter(tokenVerifier tokenPort.Verifier) *TokenVerifierAdapter {
	return &TokenVerifierAdapter{
		tokenVerifier: tokenVerifier,
	}
}

// VerifyAccessToken 验证访问令牌
// 从 token 模块的 TokenClaims 中提取用户ID、账户ID等信息
func (a *TokenVerifierAdapter) VerifyAccessToken(ctx context.Context, tokenValue string) (userID, accountID, tenantID meta.ID, err error) {
	// 调用 token 模块的验证服务
	claims, err := a.tokenVerifier.VerifyAccessToken(ctx, tokenValue)
	if err != nil {
		return meta.NewID(0), meta.NewID(0), meta.NewID(0), fmt.Errorf("token verification failed: %w", err)
	}

	// 从 claims 中提取信息
	userID = claims.UserID
	accountID = claims.AccountID

	// tenantID 目前从 token claims 中无法获取，返回 nil
	// 如果需要，可以在 TokenClaims 中添加 TenantID 字段
	return userID, accountID, meta.NewID(0), nil
}
