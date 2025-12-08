package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	authnv1 "github.com/FangcunMount/iam-contracts/api/grpc/iam/authn/v1"
	"github.com/FangcunMount/iam-contracts/pkg/sdk/config"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// =============================================================================
// JWKSManager - 使用职责链模式的管理器
// =============================================================================

// JWKSManager JWKS 密钥管理器（使用职责链模式）
type JWKSManager struct {
	config *config.JWKSConfig
	chain  KeyFetcher // 职责链头
	cache  *CacheFetcher

	stopCh chan struct{}
}

// JWKSManagerOption JWKSManager 配置选项
type JWKSManagerOption func(*jwksManagerBuilder)

type jwksManagerBuilder struct {
	config               *config.JWKSConfig
	authClient           *Client
	cbConfig             *config.CircuitBreakerConfig
	seedData             []byte
	customChain          KeyFetcher
	enableGRPC           bool
	enableCache          bool
	enableCircuitBreaker bool
}

// WithAuthClient 设置 gRPC 客户端用于降级
func WithAuthClient(client *Client) JWKSManagerOption {
	return func(b *jwksManagerBuilder) {
		b.authClient = client
		b.enableGRPC = true
	}
}

// WithCircuitBreakerConfig 启用熔断器
func WithCircuitBreakerConfig(cfg *config.CircuitBreakerConfig) JWKSManagerOption {
	return func(b *jwksManagerBuilder) {
		b.cbConfig = cfg
		b.enableCircuitBreaker = true
	}
}

// WithSeedData 设置种子数据
func WithSeedData(data []byte) JWKSManagerOption {
	return func(b *jwksManagerBuilder) { b.seedData = data }
}

// WithCustomChain 设置自定义职责链（覆盖默认链）
func WithCustomChain(chain KeyFetcher) JWKSManagerOption {
	return func(b *jwksManagerBuilder) {
		b.customChain = chain
	}
}

// WithCacheEnabled 启用缓存
func WithCacheEnabled(enabled bool) JWKSManagerOption {
	return func(b *jwksManagerBuilder) {
		b.enableCache = enabled
	}
}

// NewJWKSManager 创建 JWKS 管理器
//
// 默认职责链: Cache -> CircuitBreaker -> HTTP -> gRPC -> Seed
func NewJWKSManager(cfg *config.JWKSConfig, opts ...JWKSManagerOption) (*JWKSManager, error) {
	if cfg == nil || cfg.URL == "" {
		return nil, fmt.Errorf("jwks: url is required")
	}

	builder := &jwksManagerBuilder{
		config:      cfg,
		enableCache: true,
	}

	for _, opt := range opts {
		opt(builder)
	}

	// 构建职责链
	var chain KeyFetcher

	if builder.customChain != nil {
		chain = builder.customChain
	} else {
		chain = buildDefaultChain(builder)
	}

	// 提取缓存引用（用于后台刷新）
	var cache *CacheFetcher
	if builder.enableCache {
		if cf, ok := chain.(*CacheFetcher); ok {
			cache = cf
		}
	}

	m := &JWKSManager{
		config: cfg,
		chain:  chain,
		cache:  cache,
		stopCh: make(chan struct{}),
	}

	// 初始加载
	_, err := m.chain.Fetch(context.Background())
	if err != nil {
		return nil, fmt.Errorf("jwks: initial fetch failed: %w", err)
	}

	// 启动后台刷新
	if cfg.RefreshInterval > 0 && cache != nil {
		go m.refreshLoop()
	}

	return m, nil
}

// buildDefaultChain 构建默认职责链
//
// 链结构: Cache -> CircuitBreaker -> HTTP -> gRPC -> Seed
func buildDefaultChain(b *jwksManagerBuilder) KeyFetcher {
	// 从后往前构建链

	// 1. Seed Fetcher（最后手段）
	var tail KeyFetcher
	if len(b.seedData) > 0 {
		seedFetcher, _ := NewSeedFetcher(b.seedData)
		if seedFetcher.keySet != nil {
			tail = seedFetcher
		}
	}

	// 2. gRPC Fetcher（降级方式）
	// 优先使用传入的 authClient，其次检查配置中的 GRPCEndpoint
	if b.enableGRPC && b.authClient != nil {
		grpcFetcher := NewGRPCFetcher(b.authClient, WithGRPCNext(tail))
		tail = grpcFetcher
	} else if b.config.GRPCEndpoint != "" {
		// 配置了 GRPCEndpoint 但没有传入 authClient，创建独立的 gRPC fetcher
		grpcFetcher := NewGRPCEndpointFetcher(b.config.GRPCEndpoint, WithGRPCEndpointNext(tail))
		tail = grpcFetcher
	}

	// 3. HTTP Fetcher
	httpOpts := []HTTPFetcherOption{
		WithHTTPTimeout(b.config.RequestTimeout),
		WithHTTPNext(tail),
	}
	if b.config.HTTPClient != nil {
		httpOpts = append(httpOpts, WithHTTPClient(b.config.HTTPClient))
	}
	if len(b.config.CustomHeaders) > 0 {
		httpOpts = append(httpOpts, WithHTTPHeaders(b.config.CustomHeaders))
	}
	httpFetcher := NewHTTPFetcher(b.config.URL, httpOpts...)
	tail = httpFetcher

	// 4. CircuitBreaker Fetcher
	if b.enableCircuitBreaker {
		cbFetcher := NewCircuitBreakerFetcher(tail, b.cbConfig)
		tail = cbFetcher
	}

	// 5. Cache Fetcher（链头）
	if b.enableCache {
		cacheFetcher := NewCacheFetcher(
			WithCacheTTL(b.config.RefreshInterval),
			WithCacheNext(tail),
		)
		return cacheFetcher
	}

	return tail
}

// GetKeySet 获取当前密钥集
func (m *JWKSManager) GetKeySet(ctx context.Context) (jwk.Set, error) {
	return m.chain.Fetch(ctx)
}

// ForceRefresh 强制刷新（绕过缓存）
func (m *JWKSManager) ForceRefresh(ctx context.Context) error {
	// 如果链头是缓存，从下一个节点开始获取
	if m.cache != nil && m.cache.next != nil {
		keySet, err := m.cache.next.Fetch(ctx)
		if err != nil {
			return err
		}
		m.cache.Update(keySet)
		return nil
	}
	_, err := m.chain.Fetch(ctx)
	return err
}

// Stop 停止后台刷新
func (m *JWKSManager) Stop() {
	close(m.stopCh)
}

func (m *JWKSManager) refreshLoop() {
	ticker := time.NewTicker(m.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = m.ForceRefresh(context.Background())
		case <-m.stopCh:
			return
		}
	}
}

// =============================================================================
// 职责链模式：KeyFetcher 接口
// =============================================================================

// KeyFetcher 定义获取 JWKS 的职责链节点
type KeyFetcher interface {
	// Fetch 尝试获取密钥集，失败时调用下一个节点
	Fetch(ctx context.Context) (jwk.Set, error)
	// Name 返回 fetcher 名称（用于日志/统计）
	Name() string
}

// =============================================================================
// 1. HTTPFetcher - HTTP 方式获取 JWKS（主要方式）
// =============================================================================

// HTTPFetcher 通过 HTTP 获取 JWKS
type HTTPFetcher struct {
	url           string
	client        *http.Client
	timeout       time.Duration
	customHeaders map[string]string
	next          KeyFetcher
	stats         *FetcherStats
}

// HTTPFetcherOption HTTP Fetcher 配置选项
type HTTPFetcherOption func(*HTTPFetcher)

// WithHTTPClient 设置 HTTP 客户端
func WithHTTPClient(client *http.Client) HTTPFetcherOption {
	return func(f *HTTPFetcher) {
		f.client = client
	}
}

// WithHTTPTimeout 设置超时
func WithHTTPTimeout(timeout time.Duration) HTTPFetcherOption {
	return func(f *HTTPFetcher) {
		f.timeout = timeout
	}
}

// WithHTTPHeaders 设置自定义请求头
func WithHTTPHeaders(headers map[string]string) HTTPFetcherOption {
	return func(f *HTTPFetcher) {
		f.customHeaders = headers
	}
}

// WithHTTPNext 设置下一个 fetcher
func WithHTTPNext(next KeyFetcher) HTTPFetcherOption {
	return func(f *HTTPFetcher) {
		f.next = next
	}
}

// NewHTTPFetcher 创建 HTTP Fetcher
func NewHTTPFetcher(url string, opts ...HTTPFetcherOption) *HTTPFetcher {
	f := &HTTPFetcher{
		url:     url,
		timeout: 10 * time.Second,
		stats:   &FetcherStats{},
	}
	for _, opt := range opts {
		opt(f)
	}
	if f.client == nil {
		f.client = &http.Client{Timeout: f.timeout}
	}
	return f
}

func (f *HTTPFetcher) Name() string {
	return "http"
}

func (f *HTTPFetcher) Fetch(ctx context.Context) (jwk.Set, error) {
	f.stats.IncrAttempts()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return f.tryNext(ctx, err)
	}

	for k, v := range f.customHeaders {
		req.Header.Set(k, v)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return f.tryNext(ctx, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return f.tryNext(ctx, fmt.Errorf("http: unexpected status %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return f.tryNext(ctx, err)
	}

	keySet, err := jwk.Parse(body)
	if err != nil {
		return f.tryNext(ctx, err)
	}

	f.stats.IncrSuccesses()
	return keySet, nil
}

func (f *HTTPFetcher) tryNext(ctx context.Context, err error) (jwk.Set, error) {
	f.stats.IncrFailures()
	if f.next != nil {
		return f.next.Fetch(ctx)
	}
	return nil, fmt.Errorf("http fetcher failed: %w", err)
}

// Stats 返回统计信息
func (f *HTTPFetcher) Stats() FetcherStats {
	return f.stats.Snapshot()
}

// =============================================================================
// 2. GRPCFetcher - gRPC 方式获取 JWKS（降级方式）
// =============================================================================

// GRPCFetcher 通过 gRPC 获取 JWKS
type GRPCFetcher struct {
	client *Client
	next   KeyFetcher
	stats  *FetcherStats
}

// GRPCFetcherOption gRPC Fetcher 配置选项
type GRPCFetcherOption func(*GRPCFetcher)

// WithGRPCNext 设置下一个 fetcher
func WithGRPCNext(next KeyFetcher) GRPCFetcherOption {
	return func(f *GRPCFetcher) {
		f.next = next
	}
}

// NewGRPCFetcher 创建 gRPC Fetcher
func NewGRPCFetcher(client *Client, opts ...GRPCFetcherOption) *GRPCFetcher {
	f := &GRPCFetcher{
		client: client,
		stats:  &FetcherStats{},
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *GRPCFetcher) Name() string {
	return "grpc"
}

func (f *GRPCFetcher) Fetch(ctx context.Context) (jwk.Set, error) {
	if f.client == nil {
		return f.tryNext(ctx, fmt.Errorf("grpc: client not configured"))
	}

	f.stats.IncrAttempts()

	resp, err := f.client.GetJWKS(ctx, &authnv1.GetJWKSRequest{})
	if err != nil {
		return f.tryNext(ctx, err)
	}

	keySet, err := jwk.Parse(resp.Jwks)
	if err != nil {
		return f.tryNext(ctx, err)
	}

	f.stats.IncrSuccesses()
	return keySet, nil
}

func (f *GRPCFetcher) tryNext(ctx context.Context, err error) (jwk.Set, error) {
	f.stats.IncrFailures()
	if f.next != nil {
		return f.next.Fetch(ctx)
	}
	return nil, fmt.Errorf("grpc fetcher failed: %w", err)
}

// Stats 返回统计信息
func (f *GRPCFetcher) Stats() FetcherStats {
	return f.stats.Snapshot()
}

// =============================================================================
// 2b. GRPCEndpointFetcher - 通过 endpoint 直接连接 gRPC（无需 Client）
// =============================================================================

// GRPCEndpointFetcher 通过 endpoint 直接获取 JWKS
type GRPCEndpointFetcher struct {
	endpoint string
	conn     *grpc.ClientConn
	client   authnv1.JWKSServiceClient
	next     KeyFetcher
	stats    *FetcherStats

	mu       sync.Mutex
	initOnce sync.Once
	initErr  error
}

// GRPCEndpointFetcherOption gRPC Endpoint Fetcher 配置选项
type GRPCEndpointFetcherOption func(*GRPCEndpointFetcher)

// WithGRPCEndpointNext 设置下一个 fetcher
func WithGRPCEndpointNext(next KeyFetcher) GRPCEndpointFetcherOption {
	return func(f *GRPCEndpointFetcher) {
		f.next = next
	}
}

// NewGRPCEndpointFetcher 创建通过 endpoint 连接的 gRPC Fetcher
func NewGRPCEndpointFetcher(endpoint string, opts ...GRPCEndpointFetcherOption) *GRPCEndpointFetcher {
	f := &GRPCEndpointFetcher{
		endpoint: endpoint,
		stats:    &FetcherStats{},
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *GRPCEndpointFetcher) Name() string {
	return "grpc-endpoint"
}

func (f *GRPCEndpointFetcher) init(ctx context.Context) error {
	f.initOnce.Do(func() {
		// 使用 insecure 连接（JWKS 本身是公开数据）
		conn, err := grpc.DialContext(ctx, f.endpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			f.initErr = fmt.Errorf("grpc-endpoint: dial %s: %w", f.endpoint, err)
			return
		}
		f.conn = conn
		f.client = authnv1.NewJWKSServiceClient(conn)
	})
	return f.initErr
}

func (f *GRPCEndpointFetcher) Fetch(ctx context.Context) (jwk.Set, error) {
	f.stats.IncrAttempts()

	// 懒初始化连接
	if err := f.init(ctx); err != nil {
		return f.tryNext(ctx, err)
	}

	resp, err := f.client.GetJWKS(ctx, &authnv1.GetJWKSRequest{})
	if err != nil {
		return f.tryNext(ctx, err)
	}

	keySet, err := jwk.Parse(resp.Jwks)
	if err != nil {
		return f.tryNext(ctx, err)
	}

	f.stats.IncrSuccesses()
	return keySet, nil
}

func (f *GRPCEndpointFetcher) tryNext(ctx context.Context, err error) (jwk.Set, error) {
	f.stats.IncrFailures()
	if f.next != nil {
		return f.next.Fetch(ctx)
	}
	return nil, fmt.Errorf("grpc-endpoint fetcher failed: %w", err)
}

// Close 关闭连接
func (f *GRPCEndpointFetcher) Close() error {
	if f.conn != nil {
		return f.conn.Close()
	}
	return nil
}

// Stats 返回统计信息
func (f *GRPCEndpointFetcher) Stats() FetcherStats {
	return f.stats.Snapshot()
}

// =============================================================================
// 3. CacheFetcher - 内存缓存（快速路径）
// =============================================================================

// CacheFetcher 从内存缓存获取 JWKS
type CacheFetcher struct {
	mu      sync.RWMutex
	keySet  jwk.Set
	updated time.Time
	ttl     time.Duration
	next    KeyFetcher
	stats   *FetcherStats
}

// CacheFetcherOption 缓存 Fetcher 配置选项
type CacheFetcherOption func(*CacheFetcher)

// WithCacheTTL 设置缓存 TTL
func WithCacheTTL(ttl time.Duration) CacheFetcherOption {
	return func(f *CacheFetcher) {
		f.ttl = ttl
	}
}

// WithCacheNext 设置下一个 fetcher
func WithCacheNext(next KeyFetcher) CacheFetcherOption {
	return func(f *CacheFetcher) {
		f.next = next
	}
}

// NewCacheFetcher 创建缓存 Fetcher
func NewCacheFetcher(opts ...CacheFetcherOption) *CacheFetcher {
	f := &CacheFetcher{
		ttl:   5 * time.Minute,
		stats: &FetcherStats{},
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *CacheFetcher) Name() string {
	return "cache"
}

func (f *CacheFetcher) Fetch(ctx context.Context) (jwk.Set, error) {
	f.stats.IncrAttempts()

	f.mu.RLock()
	keySet := f.keySet
	updated := f.updated
	f.mu.RUnlock()

	// 缓存有效
	if keySet != nil && time.Since(updated) < f.ttl {
		f.stats.IncrSuccesses()
		return keySet, nil
	}

	// 缓存过期或不存在，调用下一个 fetcher
	if f.next == nil {
		f.stats.IncrFailures()
		return nil, fmt.Errorf("cache: no data and no next fetcher")
	}

	newKeySet, err := f.next.Fetch(ctx)
	if err != nil {
		// 下游失败，如果有过期缓存则降级使用
		if keySet != nil {
			f.stats.IncrSuccesses() // 降级成功
			return keySet, nil
		}
		f.stats.IncrFailures()
		return nil, err
	}

	// 更新缓存
	f.mu.Lock()
	f.keySet = newKeySet
	f.updated = time.Now()
	f.mu.Unlock()

	f.stats.IncrSuccesses()
	return newKeySet, nil
}

// Update 手动更新缓存
func (f *CacheFetcher) Update(keySet jwk.Set) {
	f.mu.Lock()
	f.keySet = keySet
	f.updated = time.Now()
	f.mu.Unlock()
}

// Stats 返回统计信息
func (f *CacheFetcher) Stats() FetcherStats {
	return f.stats.Snapshot()
}

// =============================================================================
// 4. SeedFetcher - 本地种子缓存（最后手段）
// =============================================================================

// SeedFetcher 从本地种子获取 JWKS（作为最后手段）
type SeedFetcher struct {
	keySet jwk.Set
	stats  *FetcherStats
}

// NewSeedFetcher 创建种子 Fetcher
func NewSeedFetcher(seedData []byte) (*SeedFetcher, error) {
	if len(seedData) == 0 {
		return &SeedFetcher{stats: &FetcherStats{}}, nil
	}

	keySet, err := jwk.Parse(seedData)
	if err != nil {
		return nil, fmt.Errorf("seed: parse failed: %w", err)
	}

	return &SeedFetcher{
		keySet: keySet,
		stats:  &FetcherStats{},
	}, nil
}

// NewSeedFetcherFromSet 从已解析的 KeySet 创建
func NewSeedFetcherFromSet(keySet jwk.Set) *SeedFetcher {
	return &SeedFetcher{
		keySet: keySet,
		stats:  &FetcherStats{},
	}
}

func (f *SeedFetcher) Name() string {
	return "seed"
}

func (f *SeedFetcher) Fetch(ctx context.Context) (jwk.Set, error) {
	f.stats.IncrAttempts()

	if f.keySet == nil {
		f.stats.IncrFailures()
		return nil, fmt.Errorf("seed: no seed data available")
	}

	f.stats.IncrSuccesses()
	return f.keySet, nil
}

// Stats 返回统计信息
func (f *SeedFetcher) Stats() FetcherStats {
	return f.stats.Snapshot()
}

// =============================================================================
// 5. CircuitBreakerFetcher - 熔断器包装
// =============================================================================

// CircuitBreakerFetcher 为下游 fetcher 添加熔断保护
type CircuitBreakerFetcher struct {
	next   KeyFetcher
	config *config.CircuitBreakerConfig

	mu           sync.RWMutex
	state        CircuitState
	failures     int
	successes    int
	lastFailTime time.Time
	stats        *FetcherStats
}

// CircuitState 熔断器状态
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// NewCircuitBreakerFetcher 创建熔断器 Fetcher
func NewCircuitBreakerFetcher(next KeyFetcher, cfg *config.CircuitBreakerConfig) *CircuitBreakerFetcher {
	if cfg == nil {
		cfg = &config.CircuitBreakerConfig{
			FailureThreshold: 5,
			OpenDuration:     30 * time.Second,
			HalfOpenRequests: 3,
			SuccessThreshold: 2,
		}
	}
	return &CircuitBreakerFetcher{
		next:   next,
		config: cfg,
		state:  CircuitClosed,
		stats:  &FetcherStats{},
	}
}

func (f *CircuitBreakerFetcher) Name() string {
	return "circuit-breaker"
}

func (f *CircuitBreakerFetcher) Fetch(ctx context.Context) (jwk.Set, error) {
	f.stats.IncrAttempts()

	if !f.shouldAllow() {
		f.stats.IncrFailures()
		return nil, fmt.Errorf("circuit-breaker: circuit is open")
	}

	keySet, err := f.next.Fetch(ctx)
	if err != nil {
		f.recordFailure()
		f.stats.IncrFailures()
		return nil, err
	}

	f.recordSuccess()
	f.stats.IncrSuccesses()
	return keySet, nil
}

func (f *CircuitBreakerFetcher) shouldAllow() bool {
	f.mu.RLock()
	state := f.state
	lastFail := f.lastFailTime
	f.mu.RUnlock()

	switch state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(lastFail) > f.config.OpenDuration {
			f.mu.Lock()
			f.state = CircuitHalfOpen
			f.successes = 0
			f.mu.Unlock()
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	}
	return true
}

func (f *CircuitBreakerFetcher) recordFailure() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failures++
	f.lastFailTime = time.Now()

	switch f.state {
	case CircuitClosed:
		if f.failures >= f.config.FailureThreshold {
			f.state = CircuitOpen
		}
	case CircuitHalfOpen:
		f.state = CircuitOpen
		f.failures = 0
	}
}

func (f *CircuitBreakerFetcher) recordSuccess() {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch f.state {
	case CircuitClosed:
		f.failures = 0
	case CircuitHalfOpen:
		f.successes++
		if f.successes >= f.config.SuccessThreshold {
			f.state = CircuitClosed
			f.failures = 0
		}
	}
}

// State 返回当前熔断器状态
func (f *CircuitBreakerFetcher) State() CircuitState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state
}

// Stats 返回统计信息
func (f *CircuitBreakerFetcher) Stats() FetcherStats {
	return f.stats.Snapshot()
}

// =============================================================================
// FetcherStats - 统计信息
// =============================================================================

// FetcherStats fetcher 统计信息
type FetcherStats struct {
	mu        sync.Mutex
	attempts  int64
	successes int64
	failures  int64
}

func (s *FetcherStats) IncrAttempts() {
	s.mu.Lock()
	s.attempts++
	s.mu.Unlock()
}

func (s *FetcherStats) IncrSuccesses() {
	s.mu.Lock()
	s.successes++
	s.mu.Unlock()
}

func (s *FetcherStats) IncrFailures() {
	s.mu.Lock()
	s.failures++
	s.mu.Unlock()
}

func (s *FetcherStats) Snapshot() FetcherStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return FetcherStats{
		attempts:  s.attempts,
		successes: s.successes,
		failures:  s.failures,
	}
}

// Attempts 返回尝试次数
func (s FetcherStats) Attempts() int64  { return s.attempts }
func (s FetcherStats) Successes() int64 { return s.successes }
func (s FetcherStats) Failures() int64  { return s.failures }

// =============================================================================
// JWKSStats（兼容旧接口）
// =============================================================================

// JWKSStats JWKS 统计信息
type JWKSStats struct {
	HTTPFetches   int64
	GRPCFetches   int64
	CacheHits     int64
	SeedCacheHits int64
	LastUpdate    time.Time
	State         CircuitState
}
