package authnsdk

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// JWKSManager downloads and caches IAM JWKS.
type JWKSManager struct {
	url             string
	httpClient      *http.Client
	refreshInterval time.Duration
	cacheTTL        time.Duration

	mu          sync.RWMutex
	keys        map[string]interface{}
	lastRefresh time.Time
	etag        string
}

func newJWKSManager(cfg Config) *JWKSManager {
	client := &http.Client{Timeout: cfg.JWKSRequestTimeout}
	return &JWKSManager{
		url:             cfg.JWKSURL,
		httpClient:      client,
		refreshInterval: cfg.JWKSRefreshInterval,
		cacheTTL:        cfg.JWKSCacheTTL,
	}
}

// Keyfunc returns a jwt.Keyfunc compatible with jwt Parser.
func (m *JWKSManager) Keyfunc(ctx context.Context) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		if err := m.ensureFresh(ctx); err != nil {
			return nil, err
		}
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, fmt.Errorf("token missing kid header")
		}
		rawKey, err := m.lookupKey(ctx, kid)
		if err != nil {
			return nil, err
		}
		return rawKey, nil
	}
}

// ensureFresh fetches JWKS if cache expired.
func (m *JWKSManager) ensureFresh(ctx context.Context) error {
	m.mu.RLock()
	valid := m.keys != nil && time.Since(m.lastRefresh) < m.refreshInterval
	m.mu.RUnlock()
	if valid {
		return nil
	}
	return m.Refresh(ctx)
}

// Refresh forces a JWKS download.
func (m *JWKSManager) Refresh(ctx context.Context) error {
	if m.url == "" {
		return fmt.Errorf("jwks url not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.url, nil)
	if err != nil {
		return err
	}
	m.mu.RLock()
	if m.etag != "" {
		req.Header.Set("If-None-Match", m.etag)
	}
	m.mu.RUnlock()
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		m.mu.Lock()
		m.lastRefresh = time.Now()
		m.mu.Unlock()
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("jwks fetch failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	parsedKeys, err := parseJWKSKeys(payload)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.keys = parsedKeys
	m.lastRefresh = time.Now()
	m.etag = resp.Header.Get("ETag")
	m.mu.Unlock()
	return nil
}

func (m *JWKSManager) lookupKey(ctx context.Context, kid string) (interface{}, error) {
	m.mu.RLock()
	keys := m.keys
	m.mu.RUnlock()
	if keys == nil {
		if err := m.Refresh(ctx); err != nil {
			return nil, err
		}
		m.mu.RLock()
		keys = m.keys
		m.mu.RUnlock()
	}
	if keys == nil {
		return nil, fmt.Errorf("jwks not loaded")
	}
	if key, ok := keys[kid]; ok {
		return key, nil
	}
	// retry once after refresh
	if err := m.Refresh(ctx); err != nil {
		return nil, err
	}
	m.mu.RLock()
	keys = m.keys
	m.mu.RUnlock()
	if keys == nil {
		return nil, fmt.Errorf("jwks not available after refresh")
	}
	key, ok := keys[kid]
	if !ok {
		return nil, fmt.Errorf("kid %s not found in jwks", kid)
	}
	return key, nil
}

type jwkSet struct {
	Keys []jwkEntry `json:"keys"`
}

type jwkEntry struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func parseJWKSKeys(payload []byte) (map[string]interface{}, error) {
	var set jwkSet
	if err := json.Unmarshal(payload, &set); err != nil {
		return nil, fmt.Errorf("failed to decode jwks json: %w", err)
	}
	if len(set.Keys) == 0 {
		return nil, fmt.Errorf("jwks: no keys present")
	}
	result := make(map[string]interface{}, len(set.Keys))
	for _, entry := range set.Keys {
		if entry.Kid == "" {
			continue
		}
		key, err := convertJWK(entry)
		if err != nil {
			return nil, fmt.Errorf("convert kid %s: %w", entry.Kid, err)
		}
		result[entry.Kid] = key
	}
	return result, nil
}

func convertJWK(entry jwkEntry) (interface{}, error) {
	switch entry.Kty {
	case "RSA":
		return parseRSA(entry)
	case "EC":
		return parseEC(entry)
	default:
		return nil, fmt.Errorf("unsupported kty %s", entry.Kty)
	}
}

func parseRSA(entry jwkEntry) (interface{}, error) {
	if entry.N == "" || entry.E == "" {
		return nil, fmt.Errorf("missing modulus or exponent")
	}
	modBytes, err := base64.RawURLEncoding.DecodeString(entry.N)
	if err != nil {
		return nil, fmt.Errorf("invalid modulus: %w", err)
	}
	expBytes, err := base64.RawURLEncoding.DecodeString(entry.E)
	if err != nil {
		return nil, fmt.Errorf("invalid exponent: %w", err)
	}
	e := 0
	for _, b := range expBytes {
		e = e<<8 | int(b)
	}
	if e == 0 {
		e = 65537
	}
	pub := &rsa.PublicKey{
		N: new(big.Int).SetBytes(modBytes),
		E: e,
	}
	return pub, nil
}

func parseEC(entry jwkEntry) (interface{}, error) {
	if entry.Crv == "" || entry.X == "" || entry.Y == "" {
		return nil, fmt.Errorf("missing ec parameters")
	}
	var curve elliptic.Curve
	switch entry.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported curve %s", entry.Crv)
	}
	xBytes, err := base64.RawURLEncoding.DecodeString(entry.X)
	if err != nil {
		return nil, fmt.Errorf("invalid x coordinate: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(entry.Y)
	if err != nil {
		return nil, fmt.Errorf("invalid y coordinate: %w", err)
	}
	pub := &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}
	return pub, nil
}
