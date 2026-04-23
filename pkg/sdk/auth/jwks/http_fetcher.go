package jwks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

// HTTPFetcher 通过 HTTP 获取 JWKS。
type HTTPFetcher struct {
	url           string
	client        *http.Client
	timeout       time.Duration
	customHeaders map[string]string
	next          KeyFetcher
	stats         *FetcherStats
}

// HTTPFetcherOption HTTP Fetcher 配置选项。
type HTTPFetcherOption func(*HTTPFetcher)

// WithHTTPClient 设置 HTTP 客户端。
func WithHTTPClient(client *http.Client) HTTPFetcherOption {
	return func(f *HTTPFetcher) {
		f.client = client
	}
}

// WithHTTPTimeout 设置超时。
func WithHTTPTimeout(timeout time.Duration) HTTPFetcherOption {
	return func(f *HTTPFetcher) {
		f.timeout = timeout
	}
}

// WithHTTPHeaders 设置自定义请求头。
func WithHTTPHeaders(headers map[string]string) HTTPFetcherOption {
	return func(f *HTTPFetcher) {
		f.customHeaders = headers
	}
}

// WithHTTPNext 设置下一个 fetcher。
func WithHTTPNext(next KeyFetcher) HTTPFetcherOption {
	return func(f *HTTPFetcher) {
		f.next = next
	}
}

// NewHTTPFetcher 创建 HTTP Fetcher。
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
	defer func() { _ = resp.Body.Close() }()

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

// Stats 返回统计信息。
func (f *HTTPFetcher) Stats() FetcherStats {
	return f.stats.Snapshot()
}
