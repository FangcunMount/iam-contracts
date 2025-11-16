package authnsdk

import "time"

// Config controls how the AuthN SDK connects to IAM.
type Config struct {
	// GRPCEndpoint is host:port for iam authn gRPC service.
	GRPCEndpoint string

	// JWKSURL points to IAM's /.well-known/jwks.json endpoint.
	JWKSURL string

	// JWKSRefreshInterval controls how often the JWKS cache is refreshed proactively.
	// Defaults to 5 minutes.
	JWKSRefreshInterval time.Duration

	// JWKSRequestTimeout controls the timeout when fetching JWKS through HTTP.
	// Defaults to 3 seconds.
	JWKSRequestTimeout time.Duration

	// JWKSCacheTTL is the fallback max TTL when server does not provide Cache-Control headers.
	// Defaults to 10 minutes.
	JWKSCacheTTL time.Duration

	// AllowedAudience constrains acceptable JWT audience (optional).
	AllowedAudience []string

	// AllowedIssuer constrains acceptable issuer (optional).
	AllowedIssuer string

	// ClockSkew is tolerated difference when checking exp/nbf.
	// Defaults to 60 seconds.
	ClockSkew time.Duration

	// ForceRemoteVerification means verifier will always call IAM VerifyToken RPC
	// even if local verification succeeds.
	ForceRemoteVerification bool
}

func (c *Config) setDefaults() {
	if c.JWKSRefreshInterval <= 0 {
		c.JWKSRefreshInterval = 5 * time.Minute
	}
	if c.JWKSRequestTimeout <= 0 {
		c.JWKSRequestTimeout = 3 * time.Second
	}
	if c.JWKSCacheTTL <= 0 {
		c.JWKSCacheTTL = 10 * time.Minute
	}
	if c.ClockSkew <= 0 {
		c.ClockSkew = time.Minute
	}
}
