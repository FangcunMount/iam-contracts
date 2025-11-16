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

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/golang-jwt/jwt/v4"
)

// JWKSManager JWKS 管理器
// 负责从 IAM 下载和缓存 JWKS（JSON Web Key Set）
// 提供密钥查找和自动刷新功能
type JWKSManager struct {
	url             string        // JWKS 端点 URL
	httpClient      *http.Client  // HTTP 客户端
	refreshInterval time.Duration // 刷新间隔
	cacheTTL        time.Duration // 缓存 TTL

	mu          sync.RWMutex           // 读写锁，保护下面的字段
	keys        map[string]interface{} // 密钥映射表，key 为 kid，value 为公钥
	lastRefresh time.Time              // 上次刷新时间
	etag        string                 // HTTP ETag，用于缓存控制
}

// newJWKSManager 创建 JWKS 管理器
// 根据配置初始化 JWKS 管理器实例
//
// 参数：
//   - cfg: SDK 配置
//
// 返回：
//   - *JWKSManager: JWKS 管理器实例
func newJWKSManager(cfg Config) *JWKSManager {
	client := &http.Client{Timeout: cfg.JWKSRequestTimeout}
	log.Infof("[AuthN SDK] Initializing JWKS manager with URL: %s, refresh interval: %v", cfg.JWKSURL, cfg.JWKSRefreshInterval)
	return &JWKSManager{
		url:             cfg.JWKSURL,
		httpClient:      client,
		refreshInterval: cfg.JWKSRefreshInterval,
		cacheTTL:        cfg.JWKSCacheTTL,
	}
}

// Keyfunc 返回密钥查找函数
// 返回一个与 jwt.Parser 兼容的密钥查找函数
// 该函数从 JWT token 的 header 中提取 kid，然后查找对应的公钥
//
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - jwt.Keyfunc: JWT 密钥查找函数
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

// ensureFresh 确保 JWKS 缓存是新鲜的
// 如果缓存已过期，则触发刷新
//
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - error: 刷新失败时返回错误
func (m *JWKSManager) ensureFresh(ctx context.Context) error {
	m.mu.RLock()
	// 检查缓存是否有效（存在且未过期）
	valid := m.keys != nil && time.Since(m.lastRefresh) < m.refreshInterval
	m.mu.RUnlock()
	if valid {
		return nil
	}
	return m.Refresh(ctx)
}

// Refresh 强制刷新 JWKS
// 从服务器下载最新的 JWKS，支持 ETag 缓存控制
//
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - error: 刷新失败时返回错误
func (m *JWKSManager) Refresh(ctx context.Context) error {
	if m.url == "" {
		return fmt.Errorf("jwks url not configured")
	}
	log.Debugf("[AuthN SDK] Refreshing JWKS from %s", m.url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.url, nil)
	if err != nil {
		log.Errorf("[AuthN SDK] Failed to create JWKS request: %v", err)
		return err
	}
	m.mu.RLock()
	if m.etag != "" {
		req.Header.Set("If-None-Match", m.etag)
		log.Debugf("[AuthN SDK] Using ETag: %s", m.etag)
	}
	m.mu.RUnlock()
	resp, err := m.httpClient.Do(req)
	if err != nil {
		log.Errorf("[AuthN SDK] JWKS HTTP request failed: %v", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		log.Debug("[AuthN SDK] JWKS not modified (304), using cached keys")
		m.mu.Lock()
		m.lastRefresh = time.Now()
		m.mu.Unlock()
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		log.Errorf("[AuthN SDK] JWKS fetch failed with status %d: %s", resp.StatusCode, string(body))
		return fmt.Errorf("jwks fetch failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("[AuthN SDK] Failed to read JWKS response body: %v", err)
		return err
	}
	parsedKeys, err := parseJWKSKeys(payload)
	if err != nil {
		log.Errorf("[AuthN SDK] Failed to parse JWKS: %v", err)
		return err
	}

	m.mu.Lock()
	m.keys = parsedKeys
	m.lastRefresh = time.Now()
	m.etag = resp.Header.Get("ETag")
	m.mu.Unlock()
	log.Infof("[AuthN SDK] Successfully refreshed JWKS, loaded %d keys", len(parsedKeys))
	return nil
}

// lookupKey 查找指定 kid 的密钥
// 首先从缓存查找，如果未找到则刷新 JWKS 后重试一次
//
// 参数：
//   - ctx: 上下文
//   - kid: 密钥 ID
//
// 返回：
//   - interface{}: 公钥（*rsa.PublicKey 或 *ecdsa.PublicKey）
//   - error: 查找失败时返回错误
func (m *JWKSManager) lookupKey(ctx context.Context, kid string) (interface{}, error) {
	log.Debugf("[AuthN SDK] Looking up key with kid: %s", kid)
	m.mu.RLock()
	keys := m.keys
	m.mu.RUnlock()
	// 如果缓存为空，先刷新
	if keys == nil {
		log.Debug("[AuthN SDK] No keys in cache, refreshing JWKS")
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
	// 尝试从缓存获取
	if key, ok := keys[kid]; ok {
		log.Debugf("[AuthN SDK] Found key for kid: %s", kid)
		return key, nil
	}
	// 未找到时刷新后重试一次
	log.Debugf("[AuthN SDK] Key not found for kid %s, refreshing JWKS and retrying", kid)
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
		log.Warnf("[AuthN SDK] Kid %s not found in JWKS after refresh", kid)
		return nil, fmt.Errorf("kid %s not found in jwks", kid)
	}
	log.Debugf("[AuthN SDK] Found key for kid %s after refresh", kid)
	return key, nil
}

// jwkSet JWKS 响应结构
type jwkSet struct {
	Keys []jwkEntry `json:"keys"` // 密钥列表
}

// jwkEntry JWK 条目
// 表示一个 JSON Web Key
type jwkEntry struct {
	Kty string `json:"kty"` // 密钥类型：RSA, EC
	Use string `json:"use"` // 用途：sig, enc
	Kid string `json:"kid"` // 密钥 ID
	Alg string `json:"alg"` // 算法：RS256, ES256 等
	Crv string `json:"crv"` // EC 曲线：P-256, P-384, P-521
	X   string `json:"x"`   // EC X 坐标（base64url 编码）
	Y   string `json:"y"`   // EC Y 坐标（base64url 编码）
	N   string `json:"n"`   // RSA 模数（base64url 编码）
	E   string `json:"e"`   // RSA 指数（base64url 编码）
}

// parseJWKSKeys 解析 JWKS JSON
// 将 JSON 格式的 JWKS 转换为密钥映射表
//
// 参数：
//   - payload: JWKS JSON 字节数组
//
// 返回：
//   - map[string]interface{}: kid 到公钥的映射
//   - error: 解析失败时返回错误
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

// convertJWK 转换 JWK 条目为 Go 公钥对象
// 根据密钥类型（RSA 或 EC）调用相应的解析函数
//
// 参数：
//   - entry: JWK 条目
//
// 返回：
//   - interface{}: 公钥对象（*rsa.PublicKey 或 *ecdsa.PublicKey）
//   - error: 解析失败时返回错误
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

// parseRSA 解析 RSA 公钥
// 从 JWK 条目中提取 RSA 模数和指数，构造 RSA 公钥
//
// 参数：
//   - entry: JWK 条目
//
// 返回：
//   - interface{}: RSA 公钥
//   - error: 解析失败时返回错误
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

// parseEC 解析 EC（椭圆曲线）公钥
// 从 JWK 条目中提取 EC 曲线参数和坐标，构造 EC 公钥
//
// 参数：
//   - entry: JWK 条目
//
// 返回：
//   - interface{}: EC 公钥
//   - error: 解析失败时返回错误
func parseEC(entry jwkEntry) (interface{}, error) {
	if entry.Crv == "" || entry.X == "" || entry.Y == "" {
		return nil, fmt.Errorf("missing ec parameters")
	}
	// 根据曲线名称选择椭圆曲线
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
