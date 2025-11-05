package port

import "context"

// AuthProvider covers the subset of WeChat authentication APIs
// that higher layers depend on (code2Session / phone decrypt).
type AuthProvider interface {
	Code2Session(ctx context.Context, appID, appSecret, jsCode string) (Code2SessionResult, error)
	DecryptPhone(ctx context.Context, appID, appSecret, sessionKey, encryptedData, iv string) (DecryptPhoneResult, error)
}

// Code2SessionResult captures the response we care about when exchanging jsCode.
type Code2SessionResult struct {
	OpenID     string
	UnionID    string
	SessionKey string
}

// DecryptPhoneResult carries decrypted phone details.
type DecryptPhoneResult struct {
	PhoneNumber     string
	PurePhoneNumber string
	CountryCode     string
}
