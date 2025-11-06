package authentication

import (
	"context"
	"fmt"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication/port"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
)

func init() {
	// 注册认证凭据构建器
	RegisterCredentialBuilder(AuthJWTToken, newJWTTokenCredential)
}

// ====================== 认证凭据（认证所需的数据） ========================

// JWTTokenCredential JWT Token 认证凭据
type JWTTokenCredential struct {
	TenantID    *int64
	RemoteIP    string
	UserAgent   string
	AccessToken string
}

// Scenario 返回认证场景
func (c *JWTTokenCredential) Scenario() Scenario {
	return AuthJWTToken
}

// newJWTTokenCredential 构造 JWT Token 认证凭据
func newJWTTokenCredential(input AuthInput) (AuthCredential, error) {
	if input.AccessToken == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "jwt token is required for jwt token authentication")
	}
	return &JWTTokenCredential{
		TenantID:    input.TenantID,
		RemoteIP:    input.RemoteIP,
		UserAgent:   input.UserAgent,
		AccessToken: input.AccessToken,
	}, nil
}

// ================= 认证策略（执行认证的认证器） ========================

// JWTTokenAuthStrategy JWT Token 认证策略
// 用于 API 调用场景，使用 JWT 访问令牌进行认证
type JWTTokenAuthStrategy struct {
	scenario      Scenario
	tokenVerifier port.TokenVerifier
	accountRepo   port.AccountRepository
}

// 实现认证策略接口
var _ AuthStrategy = (*JWTTokenAuthStrategy)(nil)

// NewJWTTokenAuthStrategy 构造函数（注入依赖）
func NewJWTTokenAuthStrategy(
	tokenVerifier port.TokenVerifier,
	accountRepo port.AccountRepository,
) *JWTTokenAuthStrategy {
	return &JWTTokenAuthStrategy{
		scenario:      AuthJWTToken,
		tokenVerifier: tokenVerifier,
		accountRepo:   accountRepo,
	}
}

// Kind 返回认证策略类型
func (j *JWTTokenAuthStrategy) Kind() Scenario {
	return j.scenario
}

// Authenticate 执行 JWT Token 认证
// 认证流程：
// 1. 验证 JWT Token（签名、过期、黑名单）
// 2. 从 Token 中提取用户ID、账户ID
// 3. 检查账户状态（是否锁定/禁用）
// 4. 返回认证判决
func (j *JWTTokenAuthStrategy) Authenticate(ctx context.Context, credential AuthCredential) (AuthDecision, error) {
	tokenCredential, ok := credential.(*JWTTokenCredential)
	if !ok {
		return AuthDecision{}, fmt.Errorf("jwt token strategy expects *JWTTokenCredential, got %T", credential)
	}

	// Step 1: 验证 JWT Token
	userID, accountID, tenantID, err := j.tokenVerifier.VerifyAccessToken(ctx, tokenCredential.AccessToken)
	if err != nil {
		// Token 无效/过期/被撤销 - 返回业务失败
		return AuthDecision{
			OK:      false,
			ErrCode: ErrInvalidCredential, // 使用统一的凭据无效错误码
		}, nil
	}

	// Step 2: 检查账户状态
	enabled, locked, err := j.accountRepo.GetAccountStatus(ctx, accountID)
	if err != nil {
		// 系统异常
		return AuthDecision{}, fmt.Errorf("failed to get account status: %w", err)
	}

	if !enabled {
		return AuthDecision{
			OK:      false,
			ErrCode: ErrDisabled,
		}, nil
	}

	if locked {
		return AuthDecision{
			OK:      false,
			ErrCode: ErrLocked,
		}, nil
	}

	// Step 3: 认证成功，构造认证主体
	principal := &Principal{
		UserID:    userID,
		AccountID: accountID,
		TenantID:  tenantID,
		AMR:       []string{string(AMRJWTToken)}, // 记录认证方法
		Claims:    make(map[string]any),
	}

	// 可以添加额外的 claims
	principal.Claims["auth_method"] = "jwt_token"
	if tokenCredential.RemoteIP != "" {
		principal.Claims["remote_ip"] = tokenCredential.RemoteIP
	}

	return AuthDecision{
		OK:           true,
		Principal:    principal,
		CredentialID: 0, // JWT Token 认证不对应具体的凭据记录
	}, nil
}
