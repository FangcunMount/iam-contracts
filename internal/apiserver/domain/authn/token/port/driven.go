package port

import (
	"context"
	"time"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/token"
)

// ==================== Driven Ports (驱动端口) ====================
// 定义领域模型、领域服务所依赖的外部服务接口，基础设施适配层需提供实现

// ------------------- TokenStore 令牌存储端口 ----------------------

// 用于存储和管理刷新令牌，以及令牌黑名单
type TokenStore interface {
	// SaveRefreshToken 保存刷新令牌
	SaveRefreshToken(ctx context.Context, token *domain.Token) error

	// GetRefreshToken 获取刷新令牌
	//
	// 如果令牌不存在或已过期，返回 nil
	GetRefreshToken(ctx context.Context, tokenValue string) (*domain.Token, error)

	// DeleteRefreshToken 删除刷新令牌（用于撤销或刷新后删除旧令牌）
	DeleteRefreshToken(ctx context.Context, tokenValue string) error

	// AddToBlacklist 将令牌加入黑名单
	//
	// 参数:
	//   - tokenID: 令牌唯一标识
	//   - expiry: 黑名单有效期（通常设置为令牌剩余有效期）
	AddToBlacklist(ctx context.Context, tokenID string, expiry time.Duration) error

	// IsBlacklisted 检查令牌是否在黑名单中
	IsBlacklisted(ctx context.Context, tokenID string) (bool, error)
}

// ------------------- TokenGenerator 令牌生成器端口 ----------------------

// TokenGenerator 令牌生成器端口
//
// 用于生成和解析 JWT 访问令牌
type TokenGenerator interface {
	// GenerateAccessToken 生成访问令牌（JWT）
	//
	// 参数:
	//   - principal: 认证主体（包含用户ID、账户ID等）
	//   - expiresIn: 有效期
	//
	// 返回:
	//   - token: 令牌对象（包含 JWT 字符串）
	GenerateAccessToken(principal *authentication.Principal, expiresIn time.Duration) (*domain.Token, error)

	// ParseAccessToken 解析访问令牌
	//
	// 参数:
	//   - tokenValue: JWT 字符串
	//
	// 返回:
	//   - claims: 令牌声明
	//   - err: 解析错误（如签名无效、过期等）
	ParseAccessToken(tokenValue string) (*domain.TokenClaims, error)
}
