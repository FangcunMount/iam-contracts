package response

// TokenPair 令牌对（符合 API 文档）
type TokenPair struct {
	AccessToken  string `json:"accessToken"`            // 访问令牌
	TokenType    string `json:"tokenType"`              // 令牌类型（Bearer）
	ExpiresIn    int64  `json:"expiresIn"`              // 过期时间（秒）
	RefreshToken string `json:"refreshToken,omitempty"` // 刷新令牌（可选）
	JTI          string `json:"jti,omitempty"`          // JWT ID（可选）
}

// VerifyResponse 验证令牌响应（符合 API 文档）
type VerifyResponse struct {
	Claims  TokenClaims `json:"claims"`  // 令牌声明
	Header  interface{} `json:"header"`  // JWT Header
	Blocked bool        `json:"blocked"` // 是否被拉黑
}

// TokenClaims JWT 声明
type TokenClaims struct {
	Sub string `json:"sub"`           // Subject (UserID)
	AID string `json:"aid"`           // Account ID
	Aud string `json:"aud,omitempty"` // Audience
	Iss string `json:"iss"`           // Issuer
	IAT int64  `json:"iat"`           // Issued At
	Exp int64  `json:"exp"`           // Expiration
	JTI string `json:"jti"`           // JWT ID
	KID string `json:"kid,omitempty"` // Key ID
	SID string `json:"sid,omitempty"` // Session ID
}

// JWKSet JWKS 公钥集（符合 API 文档）
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// JWK JSON Web Key
type JWK struct {
	Kty string `json:"kty"`           // Key Type (RSA, EC, etc.)
	Kid string `json:"kid"`           // Key ID
	Use string `json:"use,omitempty"` // Public Key Use (sig, enc)
	Alg string `json:"alg,omitempty"` // Algorithm
	N   string `json:"n,omitempty"`   // RSA Modulus
	E   string `json:"e,omitempty"`   // RSA Exponent
	Crv string `json:"crv,omitempty"` // EC Curve
	X   string `json:"x,omitempty"`   // EC X Coordinate
	Y   string `json:"y,omitempty"`   // EC Y Coordinate
}
