package authnsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	authnv1 "github.com/FangcunMount/iam-contracts/internal/apiserver/interface/authn/grpc/pb/iam/authn/v1"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Verifier validates JWT locally and optionally confirms with IAM via gRPC.
type Verifier struct {
	cfg    Config
	jwks   *JWKSManager
	client *Client
	parser *jwt.Parser
}

// VerifyOptions controls per-call overrides.
type VerifyOptions struct {
	ForceRemote bool
}

// NewVerifier builds a verifier using config & client.
// When client is nil the verifier works purely locally.
func NewVerifier(cfg Config, client *Client) (*Verifier, error) {
	if cfg.JWKSURL == "" {
		return nil, fmt.Errorf("jwks url is required")
	}
	cfg.setDefaults()
	manager := newJWKSManager(cfg)
	parser := jwt.NewParser(jwt.WithClockSkew(cfg.ClockSkew))
	return &Verifier{
		cfg:    cfg,
		jwks:   manager,
		client: client,
		parser: parser,
	}, nil
}

// Verify validates token locally, then optionally call IAM.
func (v *Verifier) Verify(ctx context.Context, token string, opts *VerifyOptions) (*authnv1.VerifyTokenResponse, error) {
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}
	localClaims, err := v.parseLocal(ctx, token)
	if err != nil {
		return nil, err
	}
	resp := &authnv1.VerifyTokenResponse{
		Valid:  true,
		Status: authnv1.TokenStatus_TOKEN_STATUS_VALID,
		Claims: localClaims,
	}
	if v.shouldCallRemote(opts) {
		if v.client == nil {
			return nil, fmt.Errorf("auth client not configured for remote verification")
		}
		remote, err := v.client.Auth().VerifyToken(ctx, &authnv1.VerifyTokenRequest{
			AccessToken:     token,
			ForceRemote:     true,
			IncludeMetadata: true,
		})
		if err != nil {
			return nil, err
		}
		resp = remote
	}
	return resp, nil
}

func (v *Verifier) shouldCallRemote(opts *VerifyOptions) bool {
	if opts != nil && opts.ForceRemote {
		return true
	}
	return v.cfg.ForceRemoteVerification
}

func (v *Verifier) parseLocal(ctx context.Context, token string) (*authnv1.TokenClaims, error) {
	claims := jwt.MapClaims{}
	if _, err := v.parser.ParseWithClaims(token, claims, v.jwks.Keyfunc(ctx)); err != nil {
		return nil, err
	}
	if err := v.verifyAudience(claims); err != nil {
		return nil, err
	}
	if err := v.verifyIssuer(claims); err != nil {
		return nil, err
	}
	return mapClaimsToProto(claims), nil
}

func (v *Verifier) verifyAudience(claims jwt.MapClaims) error {
	if len(v.cfg.AllowedAudience) == 0 {
		return nil
	}
	for _, aud := range v.cfg.AllowedAudience {
		if claims.VerifyAudience(aud, true) {
			return nil
		}
	}
	return fmt.Errorf("audience mismatch")
}

func (v *Verifier) verifyIssuer(claims jwt.MapClaims) error {
	if v.cfg.AllowedIssuer == "" {
		return nil
	}
	if claims.VerifyIssuer(v.cfg.AllowedIssuer, true) {
		return nil
	}
	return fmt.Errorf("issuer mismatch")
}

func mapClaimsToProto(claims jwt.MapClaims) *authnv1.TokenClaims {
	accountID := claimString(claims, "account_id")
	if accountID == "" {
		accountID = claimString(claims, "acct")
	}
	aud := claimStrings(claims["aud"])
	p := &authnv1.TokenClaims{
		TokenId:   claimString(claims, "jti"),
		Subject:   claimString(claims, "sub"),
		UserId:    claimString(claims, "user_id"),
		AccountId: accountID,
		Issuer:    claimString(claims, "iss"),
		TenantId:  claimString(claims, "tenant_id"),
		Audience:  aud,
		IssuedAt:  timestampFromClaim(claims["iat"]),
		ExpiresAt: timestampFromClaim(claims["exp"]),
		Attributes: make(map[string]string),
	}
	for k, v := range claims {
		switch k {
		case "jti", "sub", "user_id", "account_id", "acct", "iss", "tenant_id", "aud", "iat", "exp", "nbf":
			continue
		default:
			if str := toString(v); str != "" {
				p.Attributes[k] = str
			}
		}
	}
	if len(p.Attributes) == 0 {
		p.Attributes = nil
	}
	if len(p.Audience) == 0 {
		p.Audience = nil
	}
	return p
}

func claimString(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key]; ok {
		return toString(val)
	}
	return ""
}

func claimStrings(v interface{}) []string {
	switch vv := v.(type) {
	case string:
		return []string{vv}
	case []interface{}:
		out := make([]string, 0, len(vv))
		for _, item := range vv {
			if s := toString(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func timestampFromClaim(v interface{}) *timestamppb.Timestamp {
	seconds := parseNumeric(v)
	if seconds == 0 {
		return nil
	}
	return timestamppb.New(time.Unix(seconds, 0))
}

func parseNumeric(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case float32:
		return int64(val)
	case int64:
		return val
	case int32:
		return int64(val)
	case int:
		return int64(val)
	case json.Number:
		if i, err := strconv.ParseInt(string(val), 10, 64); err == nil {
			return i
		}
	}
	return 0
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	case float64:
		if math.Trunc(val) == val {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case float32:
		if math.Trunc(float64(val)) == float64(val) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case int, int32, int64:
		return fmt.Sprintf("%d", val)
	case uint, uint32, uint64:
		return fmt.Sprintf("%d", val)
	default:
		return ""
	}
}
