package token

import "time"

// Capabilities 是 AuthN Token 领域对外提供的窄角色能力集合。
type Capabilities struct {
	TokenSetMinter TokenSetMinter
	Refresher      Refresher
	Verifier       Verifier
	Revoker        Revoker
}

// Dependencies 是 Token 领域服务所需的领域协作者与技术端口。
type Dependencies struct {
	Encoder               AccessTokenEncoder
	SignatureVerifier     AccessTokenSignatureVerifier
	TokenStore            Store
	SessionLoader         SessionLoader
	SessionRevoker        SessionRevoker
	SessionExtender       SessionExtender
	SessionRefreshExpirer SessionRefreshExpirer
	AdmissionPolicy       AdmissionPolicy
	LegacyContextDecoder  LegacyAuthenticationContextSnapshotDecoder
	Issuance              IssuanceConfig
	Now                   func() time.Time
}

// NewCapabilities 装配 Token 的签发、刷新、验证和撤销领域能力。
func NewCapabilities(deps Dependencies) Capabilities {
	// 创建用户令牌颁发器
	minter := newTokenSetMinter(deps.Encoder, deps.SessionRefreshExpirer, deps.Issuance, deps.Now)
	return Capabilities{
		// 创建用户令牌颁发器
		TokenSetMinter: minter,
		// 创建刷新器
		Refresher: newRefresher(
			minter,
			deps.TokenStore,
			deps.SessionLoader,
			deps.SessionRevoker,
			deps.SessionExtender,
			deps.AdmissionPolicy,
			deps.LegacyContextDecoder,
		),
		// 创建验证器
		Verifier: newVerifier(deps.SignatureVerifier, deps.TokenStore, deps.SessionLoader, deps.AdmissionPolicy),
		// 创建撤销器
		Revoker: newRevoker(deps.SignatureVerifier, deps.TokenStore, deps.SessionRevoker),
	}
}
