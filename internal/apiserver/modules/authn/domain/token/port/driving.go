package port

// ==================== Driving Ports (驱动端口) ====================
// 这些接口由领域层（领域服务）实现，供应用层调用
// 按照功能职责拆分，遵循接口隔离原则

// TokenIssuer 令牌签发端口
type TokenIssuer interface {
	// IssueToken 签发访问令牌和刷新令牌
	//
	// 参数:
	//   - userID: 用户唯一标识
	//   - expiresIn: 访问令牌有效期
	//
	// 返回:
	//   - accessToken: 访问令牌字符串
	//   - refreshToken: 刷新令牌字符串
	//   - err: 错误信息
	IssueToken(userID string, expiresIn int64) (accessToken string, refreshToken string, err error)
	IssueToken(ctx context.Context, auth *authentication.Authentication) (*authentication.TokenPair, error)

	RevokeToken(userID string) error
}

type TokenRefresher interface {
	// RefreshToken 刷新访问令牌
	//
	// 参数:
	//   - refreshToken: 刷新令牌字符串
	//
	// 返回:
	//   - newAccessToken: 新的访问令牌
	//   - newRefreshToken: 新的刷新令牌
	//   - err: 错误信息
	RefreshToken(refreshToken string) (newAccessToken string, newRefreshToken string, err error)
}

type TokenVerifier interface {
	// VerifyToken 验证访问令牌
	//
	// 参数:
	//   - accessToken: 访问令牌字符串
	//
	// 返回:
	//   - valid: 是否有效
	//   - err: 错误信息
	VerifyToken(accessToken string) (valid bool, err error)
}
