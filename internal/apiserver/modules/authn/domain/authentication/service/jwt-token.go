package service

import (
	"context"
	"fmt"

	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/authentication/port"
)

// JWTTokenAuthStrategy JWT Token 认证策略
// 用于 API 调用场景，使用 JWT 访问令牌进行认证
type JWTTokenAuthStrategy struct {
	scenario      domain.Scenario
	tokenVerifier port.TokenVerifier
	accountRepo   port.AccountRepository
}

// 实现认证策略接口
var _ domain.AuthStrategy = (*JWTTokenAuthStrategy)(nil)

// NewJWTTokenAuthStrategy 构造函数（注入依赖）
func NewJWTTokenAuthStrategy(
	tokenVerifier port.TokenVerifier,
	accountRepo port.AccountRepository,
) *JWTTokenAuthStrategy {
	return &JWTTokenAuthStrategy{
		scenario:      domain.AuthJWTToken,
		tokenVerifier: tokenVerifier,
		accountRepo:   accountRepo,
	}
}

// Kind 返回认证策略类型
func (j *JWTTokenAuthStrategy) Kind() domain.Scenario {
	return j.scenario
}

// Authenticate 执行 JWT Token 认证
// 认证流程：
// 1. 验证 JWT Token（签名、过期、黑名单）
// 2. 从 Token 中提取用户ID、账户ID
// 3. 检查账户状态（是否锁定/禁用）
// 4. 返回认证判决
func (j *JWTTokenAuthStrategy) Authenticate(ctx context.Context, in domain.AuthInput) (domain.AuthDecision, error) {
	// Step 1: 验证 JWT Token
	userID, accountID, tenantID, err := j.tokenVerifier.VerifyAccessToken(ctx, in.AccessToken)
	if err != nil {
		// Token 无效/过期/被撤销 - 返回业务失败
		return domain.AuthDecision{
			OK:      false,
			ErrCode: domain.ErrInvalidCredential, // 使用统一的凭据无效错误码
		}, nil
	}

	// Step 2: 检查账户状态
	enabled, locked, err := j.accountRepo.GetAccountStatus(ctx, accountID)
	if err != nil {
		// 系统异常
		return domain.AuthDecision{}, fmt.Errorf("failed to get account status: %w", err)
	}

	if !enabled {
		return domain.AuthDecision{
			OK:      false,
			ErrCode: domain.ErrDisabled,
		}, nil
	}

	if locked {
		return domain.AuthDecision{
			OK:      false,
			ErrCode: domain.ErrLocked,
		}, nil
	}

	// Step 3: 认证成功，构造认证主体
	principal := &domain.Principal{
		UserID:    userID,
		AccountID: accountID,
		TenantID:  tenantID,
		AMR:       []string{string(domain.AMRJWTToken)}, // 记录认证方法
		Claims:    make(map[string]any),
	}

	// 可以添加额外的 claims
	principal.Claims["auth_method"] = "jwt_token"
	if in.RemoteIP != "" {
		principal.Claims["remote_ip"] = in.RemoteIP
	}

	return domain.AuthDecision{
		OK:           true,
		Principal:    principal,
		CredentialID: 0, // JWT Token 认证不对应具体的凭据记录
	}, nil
}
