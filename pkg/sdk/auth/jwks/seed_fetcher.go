package jwks

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

// SeedFetcher 从本地种子获取 JWKS（作为最后手段）。
type SeedFetcher struct {
	keySet jwk.Set
	stats  *FetcherStats
}

// NewSeedFetcher 创建种子 Fetcher。
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

// NewSeedFetcherFromSet 从已解析的 KeySet 创建。
func NewSeedFetcherFromSet(keySet jwk.Set) *SeedFetcher {
	return &SeedFetcher{
		keySet: keySet,
		stats:  &FetcherStats{},
	}
}

func (f *SeedFetcher) Name() string {
	return "seed"
}

func (f *SeedFetcher) Fetch(context.Context) (jwk.Set, error) {
	f.stats.IncrAttempts()

	if f.keySet == nil {
		f.stats.IncrFailures()
		return nil, fmt.Errorf("seed: no seed data available")
	}

	f.stats.IncrSuccesses()
	return f.keySet, nil
}

// Stats 返回统计信息。
func (f *SeedFetcher) Stats() FetcherStats {
	return f.stats.Snapshot()
}
