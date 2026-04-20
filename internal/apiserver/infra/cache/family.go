package cache

// Family 表示 IAM 中一个稳定的缓存族标识。
type Family string

const (
	FamilyAuthnRefreshToken        Family = "authn.refresh_token"
	FamilyAuthnRevokedAccessToken  Family = "authn.revoked_access_token"
	FamilyAuthnSession             Family = "authn.session"
	FamilyAuthnUserSessionIndex    Family = "authn.user_session_index"
	FamilyAuthnAccountSessionIndex Family = "authn.account_session_index"
	FamilyAuthnLoginOTP            Family = "authn.login_otp"
	FamilyAuthnLoginOTPSendGate    Family = "authn.login_otp_send_gate"
	FamilyIDPWechatAccessToken     Family = "idp.wechat_access_token"
	FamilyIDPWechatSDK             Family = "idp.wechat_sdk"
	FamilyAuthnJWKSPublishSnapshot Family = "authn.jwks_publish_snapshot"
)

// BackendKind 表示缓存后端类型。
type BackendKind string

const (
	BackendKindRedis  BackendKind = "redis"
	BackendKindMemory BackendKind = "memory"
)

// DataRole 表示缓存族承载的数据角色。
type DataRole string

const (
	DataRoleAuthoritativeState DataRole = "authoritative_state"
	DataRoleMarkerState        DataRole = "marker_state"
	DataRoleRemoteTokenCache   DataRole = "remote_token_cache"
	DataRoleDerivedSnapshot    DataRole = "derived_snapshot"
)

// GovernanceCapability 表示第一版治理面对 family 暴露的能力。
type GovernanceCapability string

const (
	GovernanceCapabilityInspect GovernanceCapability = "inspect"
)
